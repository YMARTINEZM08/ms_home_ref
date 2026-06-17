// Package observability wires structured logging (and, later, OTel tracing/metrics).
// Phase 0 ships slog JSON logging; tracing is added in a later iteration (TODO).
package observability

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger returns a JSON slog.Logger at the given level, suitable for Cloud
// Logging ingestion (skill Rule 11).
func NewLogger(level string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
