package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// okHandler is a simple handler that always responds 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
})

func withUnit(r *http.Request, unit string) *http.Request {
	r.Header.Set("X-Unit", unit)
	return r
}

func withBearer(r *http.Request, token string) *http.Request {
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

// ---------- extractBearerToken ----------

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"no header", "", ""},
		{"valid bearer", "Bearer mytoken", "mytoken"},
		{"case insensitive", "bearer mytoken", "mytoken"},
		{"extra spaces", "Bearer  spaced ", "spaced"},
		{"missing token", "Bearer", ""},
		{"wrong scheme", "Basic abc123", ""},
		{"too many parts", "Bearer tok extra", "tok extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := extractBearerToken(req)
			if got != tt.expected {
				t.Errorf("extractBearerToken() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// ---------- resolveAuthLevel ----------

func TestResolveAuthLevel(t *testing.T) {
	levels := map[string]AuthLevel{
		"query.unit":   AuthLevelOptional,
		"forced.unit":  AuthLevelForced,
		"default.unit": AuthLevelRecommended,
	}

	tests := []struct {
		unit     string
		expected AuthLevel
	}{
		{"query.unit", AuthLevelOptional},
		{"forced.unit", AuthLevelForced},
		{"default.unit", AuthLevelRecommended},
		{"unknown.unit", AuthLevelRecommended},
		{"", AuthLevelRecommended},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got := resolveAuthLevel(tt.unit, levels)
			if got != tt.expected {
				t.Errorf("resolveAuthLevel(%q) = %v, want %v", tt.unit, got, tt.expected)
			}
		})
	}

	t.Run("nil map", func(t *testing.T) {
		got := resolveAuthLevel("anything", nil)
		if got != AuthLevelRecommended {
			t.Errorf("expected Recommended for nil map, got %v", got)
		}
	})
}

// ---------- isValidToken ----------

func TestIsValidToken(t *testing.T) {
	keys := buildKeySet([]string{"key1", "key2"})

	t.Run("valid key", func(t *testing.T) {
		if !isValidToken("key1", keys) {
			t.Error("expected key1 to be valid")
		}
	})
	t.Run("invalid key", func(t *testing.T) {
		if isValidToken("bad", keys) {
			t.Error("expected bad to be invalid")
		}
	})
	t.Run("empty key set", func(t *testing.T) {
		if isValidToken("key1", map[string]struct{}{}) {
			t.Error("expected false for empty key set")
		}
	})
}

// ---------- DefaultAuthConfig ----------

func TestDefaultAuthConfig(t *testing.T) {
	cfg := DefaultAuthConfig()

	if cfg.Enabled {
		t.Error("default config should have auth disabled")
	}

	// High-risk units must be forced.
	forced := []string{"remote.exec", "app.uninstall", "model.delete", "service.delete"}
	for _, u := range forced {
		if level, ok := cfg.UnitAuthLevels[u]; !ok || level != AuthLevelForced {
			t.Errorf("unit %q should be AuthLevelForced", u)
		}
	}

	// Query-like units must be optional.
	optional := []string{"model.list", "model.get", "engine.list", "service.list"}
	for _, u := range optional {
		if level, ok := cfg.UnitAuthLevels[u]; !ok || level != AuthLevelOptional {
			t.Errorf("unit %q should be AuthLevelOptional", u)
		}
	}
}

// ---------- Auth middleware — optional level ----------

func TestAuthOptional(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: []string{"secret"},
		UnitAuthLevels: map[string]AuthLevel{
			"query.unit": AuthLevelOptional,
		},
	}
	handler := Auth(cfg)(okHandler)

	t.Run("no token passes through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = withUnit(req, "query.unit")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("valid token passes through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = withUnit(withBearer(req, "secret"), "query.unit")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("invalid token is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = withUnit(withBearer(req, "wrong"), "query.unit")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})
}

// ---------- Auth middleware — forced level ----------

func TestAuthForced(t *testing.T) {
	cfg := AuthConfig{
		Enabled: false, // global auth disabled — forced routes still require auth
		APIKeys: []string{"secret"},
		UnitAuthLevels: map[string]AuthLevel{
			"remote.exec": AuthLevelForced,
		},
	}
	handler := Auth(cfg)(okHandler)

	t.Run("missing token rejected even when global auth disabled", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(req, "remote.exec")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"UNAUTHORIZED"`) {
			t.Errorf("expected UNAUTHORIZED in body, got %s", body)
		}
	})

	t.Run("invalid token rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(withBearer(req, "bad"), "remote.exec")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("valid token accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(withBearer(req, "secret"), "remote.exec")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("www-authenticate header present on 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(req, "remote.exec")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Header().Get("WWW-Authenticate") == "" {
			t.Error("expected WWW-Authenticate header on 401")
		}
	})
}

// ---------- Auth middleware — recommended level ----------

func TestAuthRecommended(t *testing.T) {
	t.Run("auth disabled: passes without token", func(t *testing.T) {
		cfg := AuthConfig{
			Enabled: false,
			APIKeys: []string{"secret"},
		}
		handler := Auth(cfg)(okHandler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(req, "command.unit") // not in map → recommended
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("auth enabled: rejects missing token", func(t *testing.T) {
		cfg := AuthConfig{
			Enabled: true,
			APIKeys: []string{"secret"},
		}
		handler := Auth(cfg)(okHandler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(req, "command.unit")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("auth enabled: rejects invalid token", func(t *testing.T) {
		cfg := AuthConfig{
			Enabled: true,
			APIKeys: []string{"secret"},
		}
		handler := Auth(cfg)(okHandler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(withBearer(req, "wrong"), "command.unit")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("auth enabled: accepts valid token", func(t *testing.T) {
		cfg := AuthConfig{
			Enabled: true,
			APIKeys: []string{"secret"},
		}
		handler := Auth(cfg)(okHandler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req = withUnit(withBearer(req, "secret"), "command.unit")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

// ---------- Auth middleware — no keys configured ----------

func TestAuthNoKeysConfigured(t *testing.T) {
	// With forced level and no keys, every request should be rejected (fail-secure).
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: nil,
		UnitAuthLevels: map[string]AuthLevel{
			"remote.exec": AuthLevelForced,
		},
	}
	handler := Auth(cfg)(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withUnit(withBearer(req, "anything"), "remote.exec")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when no keys configured, got %d", rec.Code)
	}
}

// ---------- Auth middleware — logging ----------

func TestAuthLogging(t *testing.T) {
	var logBuf strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	cfg := AuthConfig{
		Enabled: true,
		APIKeys: []string{"secret"},
		Logger:  logger,
	}
	handler := Auth(cfg)(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withUnit(req, "command.unit")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "auth rejected") {
		t.Errorf("expected 'auth rejected' in log output, got: %s", logOutput)
	}
}

func TestAuthNilLogger(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: []string{"secret"},
		Logger:  nil,
	}
	handler := Auth(cfg)(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withUnit(req, "command.unit")
	rec := httptest.NewRecorder()

	// Should not panic even with nil logger.
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ---------- Auth middleware — X-Unit spoofing prevention ----------

func TestAuthXUnitSpoofingPrevention(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: []string{"secret"},
		UnitAuthLevels: map[string]AuthLevel{
			"model.list": AuthLevelOptional,
		},
	}
	handler := Auth(cfg)(okHandler)

	t.Run("POST with Optional X-Unit is floored to Recommended", func(t *testing.T) {
		// Client tries to bypass auth by setting X-Unit to an Optional unit.
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", nil)
		req = withUnit(req, "model.list") // Optional level
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Should be rejected because POST floors to Recommended, requiring auth.
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 (spoofing prevented), got %d", rec.Code)
		}
	})

	t.Run("GET with Optional X-Unit is allowed", func(t *testing.T) {
		// Legitimate read-only request with Optional unit.
		req := httptest.NewRequest(http.MethodGet, "/api/v2/resources", nil)
		req = withUnit(req, "model.list")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Should pass through because GET respects Optional level.
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

// ---------- Auth middleware — multiple valid keys ----------

func TestAuthMultipleKeys(t *testing.T) {
	cfg := AuthConfig{
		Enabled: true,
		APIKeys: []string{"key1", "key2", "key3"},
		UnitAuthLevels: map[string]AuthLevel{
			"cmd": AuthLevelRecommended,
		},
	}
	handler := Auth(cfg)(okHandler)

	for _, key := range []string{"key1", "key2", "key3"} {
		t.Run("accepts "+key, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req = withUnit(withBearer(req, key), "cmd")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected 200 for key %q, got %d", key, rec.Code)
			}
		})
	}
}
