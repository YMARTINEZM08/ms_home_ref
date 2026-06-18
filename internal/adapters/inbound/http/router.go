package http

import "net/http"

// NewRouter registers HOME routes and health checks, wrapped with tracing.
// version is echoed by the health probes to identify the running revision (canary).
func NewRouter(h *Handler, version string) http.Handler {
	mux := http.NewServeMux()

	// HOME content (web "page", pocket "screen"). The {path...} wildcard captures
	// the remaining slug, mirroring digital_bff's /content/:contentType/:locale/*.
	mux.HandleFunc("GET /content/{contentType}/{locale}", h.GetContent)
	mux.HandleFunc("GET /content/{contentType}/{locale}/{path...}", h.GetContent)

	// Cloud Run health probes.
	health := healthHandler(version)
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /readyz", health)

	return traceMiddleware(mux)
}

func healthHandler(version string) http.HandlerFunc {
	body := []byte(`{"status":"ok","version":"` + version + `"}`)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
