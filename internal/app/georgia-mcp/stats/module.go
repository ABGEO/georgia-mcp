// Package stats is the bounded context of the statistics of Georgia: the tables of
// Geostat, and the inflation from its consumer price index.
package stats

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	statscache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/catalog"
	geostatadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/geostat"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	geostatclient "github.com/abgeo/georgia-mcp/pkg/geostat"
)

const switchVariable = "GEOMCP_STATS_GEOSTAT_ENABLED"

// Module holds the use cases of the context, the catalog job, and the caches. A module of
// a Geostat that is off holds none.
type Module struct {
	uses    *tool.UseCases
	holder  *catalog.Holder
	tables  *statscache.TableCache
	indices *statscache.PriceIndexCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the catalog job, the caches, and the use cases,
// unless the operator turned Geostat off. It starts the read of the catalog.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, nil)
}

// newModule builds the module. A nil transport sends each call over the network.
func newModule(cfg Config, logger *slog.Logger, transport http.RoundTripper) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "stats"))

	if !cfg.Geostat.Enabled {
		logger.Info("Geostat is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	client := geostatclient.New(cfg.Geostat.ClientOptions(transport)...)
	adapter := geostatadapter.NewAdapter(client, time.Now, geostatadapter.Pause)

	tables, err := statscache.NewTableCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the tables: %w", err)
	}

	indices, err := statscache.NewPriceIndexCache(adapter)
	if err != nil {
		_ = tables.Close()

		return nil, fmt.Errorf("building the cache of the price indices: %w", err)
	}

	holder := catalog.NewHolder(adapter, logger)
	holder.Start()

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{
			SearchTables:  app.NewSearchTables(holder),
			DescribeTable: app.NewDescribeTable(tables),
			GetCells:      app.NewGetCells(tables),
			GetInflation:  app.NewGetInflation(indices, time.Now),
		},
		holder:  holder,
		tables:  tables,
		indices: indices,
	}, nil
}

// Register adds the tools of the context. A Geostat that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the catalog job and the caches.
func (m *Module) Close() error {
	if m.uses == nil {
		return nil
	}

	return errors.Join(m.holder.Close(), m.tables.Close(), m.indices.Close())
}
