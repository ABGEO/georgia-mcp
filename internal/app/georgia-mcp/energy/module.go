package energy

import (
	"fmt"
	"log/slog"
	"time"

	energycache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/cache"
	gseadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/gse"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	gseclient "github.com/abgeo/georgia-mcp/pkg/gse"
)

const switchVariable = "GEOMCP_ENERGY_GSE_ENABLED"

// Module holds the use case of the context and its cache. A module of a GSE that is off
// holds none.
type Module struct {
	uses *tool.UseCases
	grid *energycache.GridCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use case, unless the operator
// turned GSE off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "energy"))

	if !cfg.GSE.Enabled {
		logger.Info("GSE is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	adapter := gseadapter.NewAdapter(
		gseclient.New(cfg.GSE.ClientOptions(nil, logger.With(slog.String("vendor", "GSE")))...),
		now,
	)

	grid, err := energycache.NewGridCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the grid: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{GetGrid: app.NewGetGrid(grid, now)},
		grid: grid,
	}, nil
}

// Register adds the tools of the context. A GSE that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the cache.
func (m *Module) Close() error {
	if m.uses == nil {
		return nil
	}

	return m.grid.Close()
}
