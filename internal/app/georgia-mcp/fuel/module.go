// Package fuel is the bounded context of the fuel prices: the current prices of the
// products of the fuel companies of Georgia.
package fuel

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	fuelcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/cache"
	rompetroladapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/rompetrol"
	socaradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/socar"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/tool"
	wissoladapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/wissol"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	rompetrolclient "github.com/abgeo/georgia-mcp/pkg/rompetrol"
	socarclient "github.com/abgeo/georgia-mcp/pkg/socar"
	wissolclient "github.com/abgeo/georgia-mcp/pkg/wissol"
)

const (
	wissolSwitchVariable    = "GEOMCP_FUEL_WISSOL_ENABLED"
	rompetrolSwitchVariable = "GEOMCP_FUEL_ROMPETROL_ENABLED"
	socarSwitchVariable     = "GEOMCP_FUEL_SOCAR_ENABLED"
)

// Module holds the use case of the context and the cache of each fuel company.
type Module struct {
	prices    *app.GetPrices
	companies []*fuelcache.PriceCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the clients, the adapters, the caches, and the use case. A company that
// the operator turned off is left out.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "fuel"))

	var companies []*fuelcache.PriceCache

	if cfg.Wissol.Enabled {
		wissol, err := wissolSource(cfg.Wissol, logger, now)
		if err != nil {
			return nil, err
		}

		companies = append(companies, wissol)
	} else {
		logger.Info("Wissol is off", slog.String("variable", wissolSwitchVariable))
	}

	if cfg.Rompetrol.Enabled {
		rompetrol, err := rompetrolSource(cfg.Rompetrol, logger, now)
		if err != nil {
			return nil, err
		}

		companies = append(companies, rompetrol)
	} else {
		logger.Info("Rompetrol is off", slog.String("variable", rompetrolSwitchVariable))
	}

	if cfg.SOCAR.Enabled {
		socar, err := socarSource(cfg.SOCAR, logger, now)
		if err != nil {
			return nil, err
		}

		companies = append(companies, socar)
	} else {
		logger.Info("SOCAR is off", slog.String("variable", socarSwitchVariable))
	}

	sources := make([]app.PriceSource, 0, len(companies))
	for _, company := range companies {
		sources = append(sources, company)
	}

	logger.Debug("built the context", slog.Int("companies", len(companies)))

	return &Module{prices: app.NewGetPrices(sources), companies: companies}, nil
}

func wissolSource(
	cfg WissolConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := wissolclient.New(
		wissolclient.WithBaseURL(cfg.BaseURL.String()),
		wissolclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	wissol, err := fuelcache.New(wissoladapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Wissol: %w", err)
	}

	return wissol, nil
}

func rompetrolSource(
	cfg RompetrolConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := rompetrolclient.New(
		rompetrolclient.WithBaseURL(cfg.BaseURL.String()),
		rompetrolclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	rompetrol, err := fuelcache.New(rompetroladapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Rompetrol: %w", err)
	}

	return rompetrol, nil
}

func socarSource(
	cfg SOCARConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := socarclient.New(
		socarclient.WithBaseURL(cfg.BaseURL.String()),
		socarclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	socar, err := fuelcache.New(socaradapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of SOCAR: %w", err)
	}

	return socar, nil
}

// Register adds the tool of the context. The tool needs one fuel company or more.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if len(m.companies) == 0 {
		return nil
	}

	return tool.Register(registry, m.prices)
}

// Close stops every cache.
func (m *Module) Close() error {
	errs := make([]error, 0, len(m.companies))
	for _, company := range m.companies {
		errs = append(errs, company.Close())
	}

	return errors.Join(errs...)
}
