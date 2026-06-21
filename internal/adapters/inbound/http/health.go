package handler

import "net/http"

// healthzHandler is the liveness probe. Returns 200 as long as the process
// is running. Cloud Run uses this to determine if a replica should be restarted.
func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyzHandler is the readiness probe. Returns 200 when the service is ready
// to receive traffic. Does not hard-fail on transient content-service blips —
// only on fatal configuration issues (checked at startup, not here).
func readyzHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
