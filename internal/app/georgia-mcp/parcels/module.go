package parcels

import (
	"fmt"
	"log/slog"
	"time"

	parcelcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/cache"
	napradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/napr"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	naprclient "github.com/abgeo/georgia-mcp/pkg/napr"
)

const switchVariable = "GEOMCP_PARCELS_NAPR_ENABLED"

// Module holds the use cases of the context and their cache. A module of a NAPR that is
// off holds none.
type Module struct {
	uses    *tool.UseCases
	sources *parcelcache.SourceCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use cases, unless the operator
// turned NAPR off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "parcels"))

	if !cfg.NAPR.Enabled {
		logger.Info("NAPR is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	adapter := napradapter.NewAdapter(
		naprclient.New(cfg.NAPR.ClientOptions(nil, logger.With(slog.String("vendor", "NAPR")))...),
		now,
	)

	sources, err := parcelcache.NewSourceCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the parcels: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{
			GetParcel:       app.NewGetParcel(sources),
			ListNearby:      app.NewListNearby(sources),
			SearchAddresses: app.NewSearchAddresses(sources),
		},
		sources: sources,
	}, nil
}

// Register adds the tools of the context. A NAPR that is off adds none.
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

	return m.sources.Close()
}
