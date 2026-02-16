package middleware

import (
	"net/http"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID", "X-Trace-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-Trace-ID"},
		AllowCredentials: false,
		MaxAge:           86400,
	}
}

func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			allowedOrigin := ""
			for _, o := range cfg.AllowedOrigins {
				if o == "*" || o == origin {
					allowedOrigin = origin
					break
				}
				if strings.HasPrefix(o, "*.") && strings.HasSuffix(origin, o[1:]) {
					allowedOrigin = origin
					break
				}
			}

			if allowedOrigin == "" {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", intToStr(cfg.MaxAge))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func intToStr(n int) string {
	if n <= 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}
