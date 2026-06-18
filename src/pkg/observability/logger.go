package observability

import (
	"log/slog"
	"os"
)

// NewLogger returns a structured logger.
// JSON handler in production (for Loki ingestion), text handler in development.
func NewLogger(env string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}

	if env != "production" {
		opts.Level = slog.LevelDebug
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
