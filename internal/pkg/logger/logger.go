// Package logger builds the one logger of the process.
package logger

import (
	"io"
	"log/slog"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
)

const (
	formatJSON = "json"
	formatText = "text"
)

// Config is the configuration of the logger.
type Config struct {
	Level  slog.Level `env:"LEVEL"  envDefault:"info"`
	Format string     `env:"FORMAT" envDefault:"json"`
}

// Validate checks the format.
func (c Config) Validate() error {
	if c.Format != formatJSON && c.Format != formatText {
		return &config.FieldError{Field: "Format", Problem: `must be "json" or "text"`}
	}

	return nil
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
