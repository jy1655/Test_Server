package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging middleware logs HTTP requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := newResponseWriter(w)

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log request
		duration := time.Since(start)
		log.Printf("%s %s %d %v %s",
			r.Method,
			r.RequestURI,
			wrapped.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}
