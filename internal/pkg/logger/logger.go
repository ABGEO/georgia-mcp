// Package logger builds the one logger of the process.
package logger

import (
	"io"
	"log/slog"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
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

// New returns a logger that writes to output. Every record names the service and its
// version. It adds the source position at the debug level only.
func New(cfg Config, output io.Writer, info buildinfo.Info) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.Level, AddSource: cfg.Level <= slog.LevelDebug}

	var handler slog.Handler = slog.NewJSONHandler(output, opts)
	if cfg.Format == formatText {
		handler = slog.NewTextHandler(output, opts)
	}

	return slog.New(handler).With(
		slog.String("service.name", buildinfo.Name),
		slog.String("service.version", info.Version),
	)
}
