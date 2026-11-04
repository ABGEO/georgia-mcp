package air

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	aircache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/adapter/cache"
	neaadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/adapter/nea"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	neaclient "github.com/abgeo/georgia-mcp/pkg/nea"
)

const switchVariable = "GEOMCP_AIR_NEA_ENABLED"

// Module holds the use cases of the context and their caches. A module of an NEA that is
// off holds none.
type Module struct {
	uses     *tool.UseCases
	readings *aircache.ReadingCache
	cities   *aircache.CityCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the caches, and the use cases, unless the operator
// turned the NEA off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "air"))

	if !cfg.NEA.Enabled {
		logger.Info("the NEA is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	adapter := neaadapter.NewAdapter(neaclient.New(cfg.NEA.ClientOptions(nil)...), now)

	readings, err := aircache.NewReadingCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the readings: %w", err)
	}

	cities, err := aircache.NewCityCache(adapter)
	if err != nil {
		_ = readings.Close()

		return nil, fmt.Errorf("building the cache of the cities: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{
			GetQuality:  app.NewGetQuality(readings, cities, now),
			ListHistory: app.NewListHistory(readings, cities, now),
		},
		readings: readings,
		cities:   cities,
	}, nil
}

// Register adds the tools of the context. An NEA that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the caches.
func (m *Module) Close() error {
	if m.uses == nil {
		return nil
	}

	return errors.Join(m.readings.Close(), m.cities.Close())
}
