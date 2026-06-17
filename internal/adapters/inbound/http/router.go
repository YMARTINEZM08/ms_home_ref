package http

import "net/http"

// NewRouter registers HOME routes and health checks, wrapped with tracing.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// HOME content (web "page", pocket "screen"). The {path...} wildcard captures
	// the remaining slug, mirroring digital_bff's /content/:contentType/:locale/*.
	mux.HandleFunc("GET /content/{contentType}/{locale}", h.GetContent)
	mux.HandleFunc("GET /content/{contentType}/{locale}/{path...}", h.GetContent)

	// Cloud Run health probes.
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /readyz", health)

	return traceMiddleware(mux)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
