// Package fuel is the bounded context of the fuel prices: the current prices of the
// products of the fuel companies of Georgia.
package fuel

import (
	"fmt"
	"log/slog"
	"time"

	fuelcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/cache"
	connectadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/connect"
	gulfadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/gulf"
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
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	connectclient "github.com/abgeo/georgia-mcp/pkg/connect"
	gulfclient "github.com/abgeo/georgia-mcp/pkg/gulf"
	lukoilclient "github.com/abgeo/georgia-mcp/pkg/lukoil"
	neogasclient "github.com/abgeo/georgia-mcp/pkg/neogas"
	portalclient "github.com/abgeo/georgia-mcp/pkg/portal"
	rompetrolclient "github.com/abgeo/georgia-mcp/pkg/rompetrol"
	socarclient "github.com/abgeo/georgia-mcp/pkg/socar"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
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
	gulfSwitchVariable      = "GEOMCP_FUEL_GULF_ENABLED"
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

	registry := vendors.New[*fuelcache.PriceCache](logger)
	for _, company := range companySources(cfg, logger, now) {
		registry.Add(company)
	}

	companies, err := registry.Build()
	if err != nil {
		return nil, fmt.Errorf("building the fuel companies: %w", err)
	}

	sources := make([]app.PriceSource, 0, len(companies))
	for _, company := range companies {
		sources = append(sources, company)
	}

	logger.Debug("built the context", slog.Int("companies", len(companies)))

	return &Module{prices: app.NewGetPrices(sources), companies: companies}, nil
}

// companySources lists each fuel company, with its switch, in the order of the build.
func companySources(
	cfg Config,
	logger *slog.Logger,
	now func() time.Time,
) []vendors.Entry[*fuelcache.PriceCache] {
	return []vendors.Entry[*fuelcache.PriceCache]{
		{
			Enabled: cfg.Wissol.Enabled, Name: "Wissol", Variable: wissolSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return wissolSource(cfg.Wissol, logger, now)
			},
		},
		{
			Enabled: cfg.Rompetrol.Enabled, Name: "Rompetrol", Variable: rompetrolSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return rompetrolSource(cfg.Rompetrol, logger, now)
			},
		},
		{
			Enabled: cfg.SOCAR.Enabled, Name: "SOCAR", Variable: socarSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return socarSource(cfg.SOCAR, logger, now)
			},
		},
		{
			Enabled: cfg.Connect.Enabled, Name: "Connect", Variable: connectSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return connectSource(cfg.Connect, logger, now)
			},
		},
		{
			Enabled: cfg.Lukoil.Enabled, Name: "Lukoil", Variable: lukoilSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return lukoilSource(cfg.Lukoil, logger, now)
			},
		},
		{
			Enabled: cfg.Neogas.Enabled, Name: "Neogas", Variable: neogasSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return neogasSource(cfg.Neogas, now)
			},
		},
		{
			Enabled: cfg.Portal.Enabled, Name: "Portal", Variable: portalSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return portalSource(cfg.Portal, logger, now)
			},
		},
		{
			Enabled: cfg.Gulf.Enabled, Name: "Gulf", Variable: gulfSwitchVariable,
			Build: func() (*fuelcache.PriceCache, error) {
				return gulfSource(cfg.Gulf, logger, now)
			},
		},
	}
}

func wissolSource(
	cfg vendors.Config,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := wissolclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.HTTP)),
	)

	wissol, err := fuelcache.New(wissoladapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Wissol: %w", err)
	}

	return wissol, nil
}

func rompetrolSource(
	cfg vendors.Config,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := rompetrolclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.HTTP)),
	)

	rompetrol, err := fuelcache.New(rompetroladapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Rompetrol: %w", err)
	}

	return rompetrol, nil
}

func socarSource(
	cfg vendors.Config,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := socarclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.HTTP)),
	)

	socar, err := fuelcache.New(socaradapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of SOCAR: %w", err)
	}

	return socar, nil
}

func connectSource(
	cfg vendors.Config,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := connectclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.HTTP)),
	)

	connect, err := fuelcache.New(connectadapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Connect: %w", err)
	}

	return connect, nil
}

func lukoilSource(
	cfg vendors.Config,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := lukoilclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.HTTP)),
	)

	lukoil, err := fuelcache.New(lukoiladapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Lukoil: %w", err)
	}

	return lukoil, nil
}

func neogasSource(
	cfg vendors.Config,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := neogasclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.HTTP)),
	)

	neogas, err := fuelcache.New(neogasadapter.NewAdapter(client, now))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Neogas: %w", err)
	}

	return neogas, nil
}

func portalSource(
	cfg vendors.Config,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := portalclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.HTTP)),
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
	return vendors.CloseAll(m.companies)
}

func gulfSource(
	cfg vendors.Config,
	logger *slog.Logger,
	now func() time.Time,
) (*fuelcache.PriceCache, error) {
	client := gulfclient.New(
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(
			httpclient.NewWithTransport(cfg.HTTP, gulfclient.NewTransport()),
		),
	)

	gulf, err := fuelcache.New(gulfadapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Gulf: %w", err)
	}

	return gulf, nil
}
