package bootstrap

import (
	"log/slog"

	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// modules builds every bounded context. A new context adds one line here.
func modules(_ Config, _ *slog.Logger) ([]mcpserver.Module, error) {
	return []mcpserver.Module{}, nil
}
