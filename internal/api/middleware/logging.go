package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/nghiack7/game-ad-service/pkg/logger"
)

const limitBodySize = 10 << 10 // 10KB

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" || r.Method == "DELETE" {
				// read the body
				body, err := io.ReadAll(r.Body)
				if err != nil {
					log.Errorf("Error reading body: %v", err)
				}
				defer r.Body.Close()

				if len(body) > limitBodySize {
					body = body[:limitBodySize]
				}
				r.Body = io.NopCloser(bytes.NewBuffer(body))
			}

			// Process the request
			next.ServeHTTP(rw, r)

			// Log the request
			duration := time.Since(start)
			fields := logger.Fields{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status_code": rw.statusCode,
				"duration_ms": duration.Milliseconds(),
				"user_agent":  r.UserAgent(),
				"remote_addr": r.RemoteAddr,
			}
			log.WithFields(fields).Infof("HTTP request")
		})
	}
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
