package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// AuthLevel defines how strictly authentication is enforced.
type AuthLevel int

const (
	// AuthLevelOptional allows unauthenticated access; if a token is provided it must be valid.
	AuthLevelOptional AuthLevel = iota
	// AuthLevelRecommended requires authentication when auth is enabled globally.
	AuthLevelRecommended
	// AuthLevelForced always requires a valid token regardless of global auth config.
	AuthLevelForced
)

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	// Enabled controls whether authentication is enforced for Recommended routes.
	// Forced routes always require auth; Optional routes never block on missing token.
	Enabled bool

	// APIKeys is the set of valid Bearer tokens.
	APIKeys []string

	// Logger is used for logging auth events. May be nil.
	Logger *slog.Logger

	// UnitAuthLevels maps unit names to their auth level. When a route matches a
	// unit whose name is in this map, that level is used. Unknown units fall back
	// to AuthLevelRecommended.
	UnitAuthLevels map[string]AuthLevel
}

// DefaultAuthConfig returns a sensible default: auth disabled, no keys, and the
// high-risk units marked as forced.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Enabled: false,
		APIKeys: nil,
		UnitAuthLevels: map[string]AuthLevel{
			// Queries are optional — read-only, low risk.
			"model.list":      AuthLevelOptional,
			"model.get":       AuthLevelOptional,
			"device.info":     AuthLevelOptional,
			"engine.list":     AuthLevelOptional,
			"engine.get":      AuthLevelOptional,
			"resource.status": AuthLevelOptional,
			"service.list":    AuthLevelOptional,
			"service.get":     AuthLevelOptional,
			"app.list":        AuthLevelOptional,
			"app.get":         AuthLevelOptional,

			// High-risk mutations are forced regardless of the global Enabled flag.
			"remote.exec":    AuthLevelForced,
			"app.uninstall":  AuthLevelForced,
			"model.delete":   AuthLevelForced,
			"service.delete": AuthLevelForced,
		},
	}
}

// Auth returns a middleware that enforces API-key authentication according to cfg.
//
// The middleware inspects the "Authorization: Bearer <token>" header.
//
// Decision logic per request:
//
//	level = cfg.UnitAuthLevels[unit] (defaults to AuthLevelRecommended)
//	optional  → pass through; if token present it must still be valid
//	forced    → always validate; reject if missing or invalid
//	recommended → validate only when cfg.Enabled == true
//
// The unit name is read from the "X-Unit" request header.  For write operations
// (POST/PUT/DELETE/PATCH), the auth level is floored at AuthLevelRecommended so
// that a client cannot spoof X-Unit to downgrade a mutation to Optional.  The
// X-Unit header may still upgrade the level (e.g., to Forced for remote.exec).
// For GET requests, X-Unit is used as-is.  If the unit cannot be determined, the
// request falls back to AuthLevelRecommended.
func Auth(cfg AuthConfig) func(http.Handler) http.Handler {
	validKeys := buildKeySet(cfg.APIKeys)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			unit := r.Header.Get("X-Unit")
			level := resolveAuthLevel(unit, cfg.UnitAuthLevels)

			// Security: write operations have a floor of AuthLevelRecommended.
			// This prevents clients from spoofing X-Unit to downgrade mutations
			// (e.g., /api/v2/execute running remote.exec) to Optional.
			if isWriteMethod(r.Method) && level < AuthLevelRecommended {
				level = AuthLevelRecommended
			}

			token := extractBearerToken(r)

			switch level {
			case AuthLevelOptional:
				// If no token is provided, let the request through.
				if token == "" {
					next.ServeHTTP(w, r)
					return
				}
				// If a token IS provided, it must be valid.
				if !isValidToken(token, validKeys) {
					logUnauthorized(cfg.Logger, r, unit, "invalid token on optional route")
					writeAuthError(w, "invalid API key")
					return
				}
				next.ServeHTTP(w, r)

			case AuthLevelForced:
				// Always require a valid token.
				if token == "" {
					logUnauthorized(cfg.Logger, r, unit, "missing token on forced route")
					writeAuthError(w, "authentication required")
					return
				}
				if !isValidToken(token, validKeys) {
					logUnauthorized(cfg.Logger, r, unit, "invalid token on forced route")
					writeAuthError(w, "invalid API key")
					return
				}
				next.ServeHTTP(w, r)

			default: // AuthLevelRecommended
				if !cfg.Enabled {
					// Auth is globally disabled; pass through.
					next.ServeHTTP(w, r)
					return
				}
				if token == "" {
					logUnauthorized(cfg.Logger, r, unit, "missing token")
					writeAuthError(w, "authentication required")
					return
				}
				if !isValidToken(token, validKeys) {
					logUnauthorized(cfg.Logger, r, unit, "invalid token")
					writeAuthError(w, "invalid API key")
					return
				}
				next.ServeHTTP(w, r)
			}
		})
	}
}

// buildKeySet converts a slice of API keys into a set for O(1) lookups.
func buildKeySet(keys []string) map[string]struct{} {
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k != "" {
			set[k] = struct{}{}
		}
	}
	return set
}

// resolveAuthLevel returns the AuthLevel for the given unit name. Unknown units
// default to AuthLevelRecommended.
func resolveAuthLevel(unit string, levels map[string]AuthLevel) AuthLevel {
	if levels == nil {
		return AuthLevelRecommended
	}
	if level, ok := levels[unit]; ok {
		return level
	}
	return AuthLevelRecommended
}

// isWriteMethod returns true for HTTP methods that represent mutations.
func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}

// extractBearerToken extracts the token from "Authorization: Bearer <token>".
// Returns an empty string if the header is absent or malformed.
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// isValidToken returns true if token is in the valid key set.
func isValidToken(token string, validKeys map[string]struct{}) bool {
	if len(validKeys) == 0 {
		// No keys configured — treat all tokens as invalid (fail-secure).
		return false
	}
	_, ok := validKeys[token]
	return ok
}

// logUnauthorized logs an auth failure at warn level.
func logUnauthorized(logger *slog.Logger, r *http.Request, unit, reason string) {
	if logger == nil {
		return
	}
	logger.Warn("auth rejected",
		slog.String("reason", reason),
		slog.String("unit", unit),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.String("remote_addr", r.RemoteAddr),
	)
}

// writeAuthError writes a 401 JSON response.
func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="AIMA API"`)
	w.WriteHeader(http.StatusUnauthorized)

	resp := map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}
