package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/nghiack7/game-ad-service/pkg/logger"
)

// RecoveryMiddleware recovers from panics and logs the error
func RecoveryMiddleware(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the error and stack trace
					stackTrace := debug.Stack()
					log.WithFields(logger.Fields{
						"error":       err,
						"stack_trace": string(stackTrace),
						"path":        r.URL.Path,
						"method":      r.Method,
					}).Errorf("Panic recovered")

					// Return a 500 Internal Server Error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"success":false,"error":"Internal Server Error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
