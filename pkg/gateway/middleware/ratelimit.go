package middleware

import (
	"net/http"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/ratelimit"
)

// RateLimit returns a middleware that enforces per-IP rate limiting using the
// provided Limiter. When the limit is exceeded, the middleware responds with
// HTTP 429 Too Many Requests and does not call the next handler.
//
// The rate-limit key is the request's RemoteAddr (includes port on most
// platforms). In production behind a reverse proxy, consider using the
// X-Forwarded-For or X-Real-IP header as the key instead.
func RateLimit(limiter ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if key == "" {
				key = "unknown"
			}

			allowed, err := limiter.Allow(key)
			if err != nil || !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"too many requests"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
