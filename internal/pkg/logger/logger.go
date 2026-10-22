// Package logger builds the one logger of the process.
package logger

import (
	"io"
	"log/slog"
)

const (
	formatJSON = "json"
	formatText = "text"
)

// Config is the configuration of the logger.
type Config struct {
	Level  slog.Level `env:"LEVEL"  envDefault:"info"`
	Format string     `env:"FORMAT" envDefault:"json" validate:"oneof=json text"`
}

// New returns a logger that writes to output. It adds the source position at the
// debug level only.
func New(cfg Config, output io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.Level, AddSource: cfg.Level <= slog.LevelDebug}

	if cfg.Format == formatText {
		return slog.New(slog.NewTextHandler(output, opts))
	}

	return slog.New(slog.NewJSONHandler(output, opts))
}
