package httputil

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Flush forwards to the underlying ResponseWriter's Flusher, if any, so
// wrapping in statusRecorder doesn't break SSE handlers that type-assert
// http.Flusher (embedding http.ResponseWriter alone does not promote Flush,
// since the interface itself doesn't declare it).
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// LogRoutes wraps next with a middleware that logs every request: method,
// path, resulting status code, and duration.
func LogRoutes(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
