package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					stack := debug.Stack()

					err, ok := rvr.(error)
					var errMsg string
					if ok {
						errMsg = err.Error()
					} else {
						errMsg = fmt.Sprintf("%v", rvr)
					}

					if logger != nil {
						logger.Error("panic recovered",
							slog.String("error", errMsg),
							slog.String("stack", string(stack)),
							slog.String("path", r.URL.Path),
							slog.String("method", r.Method),
						)
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					resp := map[string]any{
						"success": false,
						"error": map[string]any{
							"code":    "INTERNAL_ERROR",
							"message": "Internal server error",
						},
					}
					_ = json.NewEncoder(w).Encode(resp)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
