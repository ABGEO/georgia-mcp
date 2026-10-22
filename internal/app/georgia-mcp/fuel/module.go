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
			Name: "Wissol", Variable: wissolSwitchVariable, Config: cfg.Wissol,
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return wissoladapter.NewAdapter(wissolclient.New(opts...), now, logger)
			}),
		},
		{
			Name: "Rompetrol", Variable: rompetrolSwitchVariable, Config: cfg.Rompetrol,
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return rompetroladapter.NewAdapter(rompetrolclient.New(opts...), now, logger)
			}),
		},
		{
			Name: "SOCAR", Variable: socarSwitchVariable, Config: cfg.SOCAR,
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return socaradapter.NewAdapter(socarclient.New(opts...), now, logger)
			}),
		},
		{
			Name: "Connect", Variable: connectSwitchVariable, Config: cfg.Connect,
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return connectadapter.NewAdapter(connectclient.New(opts...), now, logger)
			}),
		},
		{
			Name: "Lukoil", Variable: lukoilSwitchVariable, Config: cfg.Lukoil,
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return lukoiladapter.NewAdapter(lukoilclient.New(opts...), now, logger)
			}),
		},
		{
			Name: "Neogas", Variable: neogasSwitchVariable, Config: cfg.Neogas,
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return neogasadapter.NewAdapter(neogasclient.New(opts...), now)
			}),
		},
		{
			Name: "Portal", Variable: portalSwitchVariable, Config: cfg.Portal,
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return portaladapter.NewAdapter(portalclient.New(opts...), now, logger)
			}),
		},
		{
			Name: "Gulf", Variable: gulfSwitchVariable, Config: cfg.Gulf,
			Transport: gulfclient.NewTransport(),
			Build: cached(func(opts ...vendorhttp.Option) app.PriceSource {
				return gulfadapter.NewAdapter(gulfclient.New(opts...), now, logger)
			}),
		},
	}
}

// cached returns the Build of a company: its adapter, in a price cache.
func cached(
	adapter func(opts ...vendorhttp.Option) app.PriceSource,
) func(opts ...vendorhttp.Option) (*fuelcache.PriceCache, error) {
	return func(opts ...vendorhttp.Option) (*fuelcache.PriceCache, error) {
		return fuelcache.New(adapter(opts...))
	}
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
