package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// modules builds every bounded context. A new context adds one line here.
func modules(cfg Config, logger *slog.Logger) ([]mcpserver.Module, error) {
	ratesModule, err := rates.New(cfg.Rates, logger)
	if err != nil {
		return nil, fmt.Errorf("building the rates context: %w", err)
	}

	return []mcpserver.Module{ratesModule}, nil
}
