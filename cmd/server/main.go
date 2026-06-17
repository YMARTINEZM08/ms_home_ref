// Command server is the ms_home entrypoint. Stateless, env-configured, with
// graceful shutdown for Cloud Run SIGTERM (skill Rule 5).
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ms_home/internal/bootstrap"
	"ms_home/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Logger not built yet; fail fast on misconfiguration.
		panic(err)
	}

	app := bootstrap.New(cfg)
	log := app.Logger

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		log.Info("server starting", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err.Error())
		os.Exit(1)
	}
	log.Info("server stopped")
}
