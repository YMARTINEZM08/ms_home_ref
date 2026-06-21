package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a JSON-structured slog.Logger configured for cloud environments.
// The level is applied via a LevelVar so it can be changed at runtime without
// redeployment (e.g., temporarily enable DEBUG during an incident).
func New(level string) *slog.Logger {
	var lvl slog.LevelVar
	lvl.Set(parseLevel(level))
	opts := &slog.HandlerOptions{Level: &lvl}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// parseLevel maps the LOG_LEVEL string to a slog.Level.
// TRACE is mapped to DEBUG (Go's slog has no TRACE level).
// OFF is mapped to a high numeric value that disables all log output.
func parseLevel(level string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG", "TRACE":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	case "OFF":
		return slog.Level(1000)
	default:
		return slog.LevelInfo
	}
}
