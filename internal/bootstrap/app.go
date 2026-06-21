package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	inbound "github.com/YMARTINEZM08/ms_home_ref/internal/adapters/inbound/http"
	"github.com/YMARTINEZM08/ms_home_ref/internal/adapters/outbound/contentservice"
	"github.com/YMARTINEZM08/ms_home_ref/internal/application/blocks"
	apphome "github.com/YMARTINEZM08/ms_home_ref/internal/application/home"
	"github.com/YMARTINEZM08/ms_home_ref/internal/config"
	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/breaker"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/httpx"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/logger"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/observability"
)

// Run is the composition root: it loads config, wires all dependencies, starts
// the HTTP server, and handles graceful shutdown on SIGTERM/SIGINT.
func Run() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error — cannot start", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	shutdownOTEL, err := observability.Init(ctx, cfg.ServiceName, cfg.OTEL.Endpoint, cfg.Environment, cfg.OTEL.SampleRatio)
	if err != nil {
		log.Warn("OTel init failed — continuing without tracing", "error", err)
		shutdownOTEL = func(_ context.Context) error { return nil }
	}

	// ── Outbound layer ───────────────────────────────────────────────────────
	httpClient := httpx.NewClient(cfg.ContentService.Timeout)

	csClient := contentservice.NewClient(
		contentservice.Config{
			BaseURL: cfg.ContentService.URL,
			Timeout: cfg.ContentService.Timeout,
			BreakerSettings: breaker.Settings{
				FailureRatio: cfg.Breaker.FailureRatio,
				MinRequests:  cfg.Breaker.MinRequests,
				OpenTimeout:  cfg.Breaker.OpenTimeout,
			},
		},
		httpClient,
		log,
	)

	// ── Application layer ────────────────────────────────────────────────────
	homeService := apphome.NewService(csClient, log)

	blockRegistry := blocks.NewRegistry(log)
	// Register a StubResolver for every dynamic block type so the service is
	// runnable end-to-end before real downstream adapters are wired.
	// Replace each StubResolver with a real Resolver as adapters are built.
	for _, bt := range []domain.BlockType{
		domain.BlockTypeProductsList,
		domain.BlockTypeBannerProducts,
		domain.BlockTypeGreeting,
		domain.BlockTypeGuestContainer,
		domain.BlockTypeShortcuts,
		domain.BlockTypeRecommendations,
		domain.BlockTypeProductCards,
	} {
		blockRegistry.Register(bt, &blocks.StubResolver{BlockType: bt})
	}

	// ── Inbound layer ────────────────────────────────────────────────────────
	router := inbound.NewRouter(homeService, blockRegistry, log, cfg.ServiceName)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ── Start ────────────────────────────────────────────────────────────────
	go func() {
		log.Info("server starting",
			"port", cfg.Port,
			"service", cfg.ServiceName,
			"environment", cfg.Environment,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info("shutdown signal received — draining connections")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", "error", err)
	}
	if err := shutdownOTEL(shutdownCtx); err != nil {
		log.Error("OTel shutdown error", "error", err)
	}

	log.Info("server stopped")
}
