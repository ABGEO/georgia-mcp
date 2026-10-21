// Package fuel is the bounded context of the fuel prices: the current prices of the
// products of the fuel companies of Georgia.
package fuel

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	fuelcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/cache"
	connectadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/connect"
	lukoiladapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/lukoil"
	neogasadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/neogas"
	portaladapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/portal"
	rompetroladapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/rompetrol"
	socaradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/socar"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/tool"
	wissoladapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/wissol"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	connectclient "github.com/abgeo/georgia-mcp/pkg/connect"
	lukoilclient "github.com/abgeo/georgia-mcp/pkg/lukoil"
	neogasclient "github.com/abgeo/georgia-mcp/pkg/neogas"
	portalclient "github.com/abgeo/georgia-mcp/pkg/portal"
	rompetrolclient "github.com/abgeo/georgia-mcp/pkg/rompetrol"
	socarclient "github.com/abgeo/georgia-mcp/pkg/socar"
	wissolclient "github.com/abgeo/georgia-mcp/pkg/wissol"
)

const (
	wissolSwitchVariable    = "GEOMCP_FUEL_WISSOL_ENABLED"
	rompetrolSwitchVariable = "GEOMCP_FUEL_ROMPETROL_ENABLED"
	socarSwitchVariable     = "GEOMCP_FUEL_SOCAR_ENABLED"
	connectSwitchVariable   = "GEOMCP_FUEL_CONNECT_ENABLED"
	lukoilSwitchVariable    = "GEOMCP_FUEL_LUKOIL_ENABLED"
	neogasSwitchVariable    = "GEOMCP_FUEL_NEOGAS_ENABLED"
	portalSwitchVariable    = "GEOMCP_FUEL_PORTAL_ENABLED"
)

// companySource is one fuel company of the configuration: its switch, and the builder of
// its cache.
type companySource struct {
	enabled  bool
	name     string
	variable string
	build    func() (*fuelcache.PriceCache, error)
}

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

	for _, source := range companySources(cfg, logger, now) {
		if !source.enabled {
			logger.Info(source.name+" is off", slog.String("variable", source.variable))

			continue
		}

		company, err := source.build()
		if err != nil {
			return nil, err
		}

		companies = append(companies, company)
	}

	sources := make([]app.PriceSource, 0, len(companies))
	for _, company := range companies {
		sources = append(sources, company)
	}

	logger.Debug("built the context", slog.Int("companies", len(companies)))

	return &Module{prices: app.NewGetPrices(sources), companies: companies}, nil
}

// companySources lists each fuel company in the order of the build.
func companySources(
	cfg Config,
	logger *slog.Logger,
	now func() time.Time,
) []companySource {
	return []companySource{
		{
			enabled: cfg.Wissol.Enabled, name: "Wissol", variable: wissolSwitchVariable,
			build: func() (*fuelcache.PriceCache, error) {
				return wissolSource(cfg.Wissol, logger, now)
			},
		},
		{
			enabled: cfg.Rompetrol.Enabled, name: "Rompetrol", variable: rompetrolSwitchVariable,
			build: func() (*fuelcache.PriceCache, error) {
				return rompetrolSource(cfg.Rompetrol, logger, now)
			},
		},
		{
			enabled: cfg.SOCAR.Enabled, name: "SOCAR", variable: socarSwitchVariable,
			build: func() (*fuelcache.PriceCache, error) {
				return socarSource(cfg.SOCAR, logger, now)
			},
		},
		{
			enabled: cfg.Connect.Enabled, name: "Connect", variable: connectSwitchVariable,
			build: func() (*fuelcache.PriceCache, error) {
				return connectSource(cfg.Connect, logger, now)
			},
		},
		{
			enabled: cfg.Lukoil.Enabled, name: "Lukoil", variable: lukoilSwitchVariable,
			build: func() (*fuelcache.PriceCache, error) {
				return lukoilSource(cfg.Lukoil, logger, now)
			},
		},
		{
			enabled: cfg.Neogas.Enabled, name: "Neogas", variable: neogasSwitchVariable,
			build: func() (*fuelcache.PriceCache, error) {
				return neogasSource(cfg.Neogas, now)
			},
		},
		{
			enabled: cfg.Portal.Enabled, name: "Portal", variable: portalSwitchVariable,
			build: func() (*fuelcache.PriceCache, error) {
				return portalSource(cfg.Portal, logger, now)
			},
		},
	}
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

func connectSource(
	cfg ConnectConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := connectclient.New(
		connectclient.WithBaseURL(cfg.BaseURL.String()),
		connectclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	connect, err := fuelcache.New(connectadapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Connect: %w", err)
	}

	return connect, nil
}

func lukoilSource(
	cfg LukoilConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := lukoilclient.New(
		lukoilclient.WithBaseURL(cfg.BaseURL.String()),
		lukoilclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	lukoil, err := fuelcache.New(lukoiladapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Lukoil: %w", err)
	}

	return lukoil, nil
}

func neogasSource(
	cfg NeogasConfig,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := neogasclient.New(
		neogasclient.WithBaseURL(cfg.BaseURL.String()),
		neogasclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	neogas, err := fuelcache.New(neogasadapter.NewAdapter(client, now))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Neogas: %w", err)
	}

	return neogas, nil
}

func portalSource(
	cfg PortalConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := portalclient.New(
		portalclient.WithBaseURL(cfg.BaseURL.String()),
		portalclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	portal, err := fuelcache.New(portaladapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Portal: %w", err)
	}

	return portal, nil
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
