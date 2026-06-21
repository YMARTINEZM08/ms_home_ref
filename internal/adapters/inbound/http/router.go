package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/YMARTINEZM08/ms_home_ref/internal/application/blocks"
	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// NewRouter wires the Chi router with middleware and all routes.
// Dependencies are passed in — the router has no global state.
func NewRouter(
	homeUC domain.HomeUseCase,
	blockRegistry blocks.ResolveUseCase,
	log *slog.Logger,
	serviceName string,
) http.Handler {
	r := chi.NewRouter()

	// ── Middleware (applied in declaration order) ───────────────────────────
	r.Use(requestIDMiddleware)
	r.Use(correlationIDMiddleware)
	// OTel spans wrap each request; named after the service for trace grouping.
	r.Use(otelhttp.NewMiddleware(serviceName))
	// Panic recovery: prevents a handler bug from crashing the process.
	r.Use(middleware.Recoverer)
	r.Use(accessLogMiddleware(log))

	// ── Routes ──────────────────────────────────────────────────────────────
	r.Get("/home", newHomeHandler(homeUC, log).ServeHTTP)
	r.Get("/home/blocks/{blockType}", newBlockHandler(blockRegistry, log).ServeHTTP)

	// Health probes — always last, not wrapped by access log.
	r.Get("/healthz", healthzHandler)
	r.Get("/readyz", readyzHandler)

	return r
}
