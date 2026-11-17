package energy

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	energycache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/cache"
	gnercadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/gnerc"
	gseadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/gse"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	gnercclient "github.com/abgeo/georgia-mcp/pkg/gnerc"
	gseclient "github.com/abgeo/georgia-mcp/pkg/gse"
)

const (
	gseSwitch   = "GEOMCP_ENERGY_GSE_ENABLED"
	gnercSwitch = "GEOMCP_ENERGY_GNERC_ENABLED"
)

// Module holds the use cases of the context and their caches. A vendor that is off adds
// no use case and no cache.
type Module struct {
	uses  tool.UseCases
	grid  *energycache.GridCache
	books *energycache.BookCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use cases of each vendor that the
// operator did not turn off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "energy"))
	module := &Module{}

	if cfg.GSE.Enabled {
		adapter := gseadapter.NewAdapter(
			gseclient.New(cfg.GSE.ClientOptions(nil, logger.With(slog.String("vendor", "GSE")))...),
			now,
		)

		grid, err := energycache.NewGridCache(adapter)
		if err != nil {
			return nil, fmt.Errorf("building the cache of the grid: %w", err)
		}

		module.grid = grid
		module.uses.GetGrid = app.NewGetGrid(grid, now)
	} else {
		logger.Info("GSE is off", slog.String("variable", gseSwitch))
	}

	if cfg.GNERC.Enabled {
		options := cfg.GNERC.ClientOptions(
			gnercclient.NewTransport(), logger.With(slog.String("vendor", "GNERC")),
		)
		adapter := gnercadapter.NewAdapter(gnercclient.New(options...), now)

		books, err := energycache.NewBookCache(adapter, adapter)
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf("building the cache of GNERC: %w", err),
				module.Close(),
			)
		}

		module.books = books
		module.uses.GetBalance = app.NewGetBalance(books)
		module.uses.ListPrices = app.NewListPrices(books)
	} else {
		logger.Info("GNERC is off", slog.String("variable", gnercSwitch))
	}

	logger.Debug("built the context")

	return module, nil
}

// Register adds the tool of each use case.
func (m *Module) Register(registry *mcpserver.Registry) error {
	return tool.Register(registry, m.uses)
}

// Close stops each cache.
func (m *Module) Close() error {
	var errs []error

	if m.grid != nil {
		errs = append(errs, m.grid.Close())
	}

	if m.books != nil {
		errs = append(errs, m.books.Close())
	}

	return errors.Join(errs...)
}
