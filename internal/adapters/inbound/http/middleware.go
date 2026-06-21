package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/YMARTINEZM08/ms_home_ref/pkg/httpx"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// safeCorrelationID limits correlation IDs to alphanumeric + hyphens/underscores
// to prevent log injection when the value is written to structured logs.
var safeCorrelationID = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,64}$`)

// requestIDMiddleware reads x-request-id from the inbound header or generates
// a new UUID, stores it in the context, and echoes it in the response.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("x-request-id")
		if id == "" {
			id = uuid.New().String()
		}
		ctx := httpx.WithRequestID(r.Context(), id)
		w.Header().Set("x-request-id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// correlationIDMiddleware reads x-correlation-id, validates it to prevent log
// injection, stores it in context, and echoes it in the response.
func correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("x-correlation-id")
		if id == "" || !safeCorrelationID.MatchString(id) {
			id = middleware.GetReqID(r.Context()) // fall back to chi's request ID
			if id == "" {
				id = uuid.New().String()
			}
		}
		ctx := httpx.WithCorrelationID(r.Context(), id)
		w.Header().Set("x-correlation-id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// accessLogMiddleware emits one structured INFO log per request containing
// method, path, status, latency, and correlation context.
// High-frequency paths (healthz/readyz) are excluded to reduce log noise.
func accessLogMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health probes — they are high-frequency and add no value.
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			log.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"latency_ms", time.Since(start).Milliseconds(),
				"request_id", httpx.RequestIDFromCtx(r.Context()),
				"correlation_id", httpx.CorrelationIDFromCtx(r.Context()),
			)
		})
	}
}
