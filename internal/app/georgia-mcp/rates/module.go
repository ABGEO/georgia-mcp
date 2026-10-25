// Package rates is the bounded context of the exchange rates: the official rates of
// the National Bank of Georgia, and the commercial rates of the banks.
package rates

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	basisbankadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/basisbank"
	bogadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/bog"
	ratecache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/cache"
	credoadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/credo"
	halykadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/halyk"
	kursiadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/kursi"
	libertyadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/liberty"
	nbgadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/nbg"
	ricoadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/rico"
	scappadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/scapp"
	tbcadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tbc"
	terabankadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/terabank"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	basisclient "github.com/abgeo/georgia-mcp/pkg/basisbank"
	bogclient "github.com/abgeo/georgia-mcp/pkg/bog"
	credoclient "github.com/abgeo/georgia-mcp/pkg/credo"
	halykclient "github.com/abgeo/georgia-mcp/pkg/halyk"
	kursiclient "github.com/abgeo/georgia-mcp/pkg/kursi"
	libertyclient "github.com/abgeo/georgia-mcp/pkg/libertyrates"
	nbgclient "github.com/abgeo/georgia-mcp/pkg/nbg"
	ricoclient "github.com/abgeo/georgia-mcp/pkg/rico"
	scappclient "github.com/abgeo/georgia-mcp/pkg/scapp"
	tbcclient "github.com/abgeo/georgia-mcp/pkg/tbc"
	terabankclient "github.com/abgeo/georgia-mcp/pkg/terabank"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	tbcSwitchVariable       = "GEOMCP_RATES_TBC_ENABLED"
	kursiSwitchVariable     = "GEOMCP_RATES_KURSI_ENABLED"
	bogSwitchVariable       = "GEOMCP_RATES_BOG_ENABLED"
	libertySwitchVariable   = "GEOMCP_RATES_LIBERTY_ENABLED"
	terabankSwitchVariable  = "GEOMCP_RATES_TERABANK_ENABLED"
	ricoSwitchVariable      = "GEOMCP_RATES_RICO_ENABLED"
	scappSwitchVariable     = "GEOMCP_RATES_SCAPP_ENABLED"
	halykSwitchVariable     = "GEOMCP_RATES_HALYK_ENABLED"
	basisbankSwitchVariable = "GEOMCP_RATES_BASISBANK_ENABLED"
	credoSwitchVariable     = "GEOMCP_RATES_CREDO_ENABLED" //nolint:gosec // A name, not a key.
)

// Module holds the use cases of the context and their caches.
type Module struct {
	getRates   *app.GetOfficialRates
	history    *app.ListOfficialHistory
	commercial *app.GetCommercialRates
	rateSets   *ratecache.RateSetCache
	banks      []*ratecache.CommercialRateCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the clients, the adapters, the caches, and the use cases. A bank that the
// operator turned off is left out.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	client := nbgclient.New(
		vendorhttp.WithBaseURL(cfg.NBG.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(cfg.NBG.HTTP)),
	)

	rateSets, err := ratecache.New(nbgadapter.NewAdapter(client, now), cfg.NBG.CacheEntries, now)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the rates: %w", err)
	}

	banks, err := commercialSources(cfg, logger, now)
	if err != nil {
		_ = rateSets.Close()

		return nil, err
	}

	sources := make([]app.CommercialRateSource, 0, len(banks))
	for _, bank := range banks {
		sources = append(sources, bank)
	}

	logger.Debug("built the context",
		slog.String("component", "context"),
		slog.String("context", "rates"),
		slog.String("nbg", cfg.NBG.BaseURL.String()),
		slog.Int("banks", len(banks)),
	)

	return &Module{
		getRates:   app.NewGetOfficialRates(rateSets, now),
		history:    app.NewListOfficialHistory(rateSets, now),
		commercial: app.NewGetCommercialRates(sources),
		rateSets:   rateSets,
		banks:      banks,
	}, nil
}

// commercialSources builds the cached source of each bank that is on.
func commercialSources(
	cfg Config,
	logger *slog.Logger,
	now func() time.Time,
) ([]*ratecache.CommercialRateCache, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "rates"))

	banks := vendors.New[*ratecache.CommercialRateCache](logger)

	for _, bank := range slices.Concat(bankEntries(cfg, logger, now), otherEntries(cfg, now)) {
		banks.Add(bank)
	}

	return banks.Build()
}

// bankEntries lists each commercial bank, with its switch, in the order of the build.
func bankEntries(
	cfg Config,
	logger *slog.Logger,
	now func() time.Time,
) []vendors.Entry[*ratecache.CommercialRateCache] {
	return []vendors.Entry[*ratecache.CommercialRateCache]{
		{
			Name: "TBC Bank", Variable: tbcSwitchVariable, Config: cfg.TBC.Config,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				client := tbcclient.New(append(opts, tbcclient.WithAPIKey(cfg.TBC.APIKey))...)

				return tbcadapter.NewAdapter(client, now)
			}),
		},
		{
			Name: "Bank of Georgia", Variable: bogSwitchVariable, Config: cfg.BOG,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return bogadapter.NewAdapter(bogclient.New(opts...), now)
			}),
		},
		{
			Name: "Liberty Bank", Variable: libertySwitchVariable, Config: cfg.Liberty,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return libertyadapter.NewAdapter(libertyclient.New(opts...), now)
			}),
		},
		{
			Name: "Terabank", Variable: terabankSwitchVariable, Config: cfg.Terabank,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return terabankadapter.NewAdapter(terabankclient.New(opts...), now)
			}),
		},
		{
			Name: "Halyk Bank", Variable: halykSwitchVariable, Config: cfg.Halyk,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return halykadapter.NewAdapter(halykclient.New(opts...), now)
			}),
		},
		{
			Name: "Basisbank", Variable: basisbankSwitchVariable, Config: cfg.Basisbank,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return basisbankadapter.NewAdapter(basisclient.New(opts...), now)
			}),
		},
		{
			Name: "Credo Bank", Variable: credoSwitchVariable, Config: cfg.Credo.Config,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				client := credoclient.New(append(opts, credoclient.WithAPIKey(cfg.Credo.APIKey))...)

				return credoadapter.NewAdapter(client, now, logger)
			}),
		},
	}
}

// otherEntries lists each microfinance organization and exchange bureau, with its
// switch, in the order of the build.
func otherEntries(
	cfg Config,
	now func() time.Time,
) []vendors.Entry[*ratecache.CommercialRateCache] {
	return []vendors.Entry[*ratecache.CommercialRateCache]{
		{
			Name: "Rico Credit", Variable: ricoSwitchVariable, Config: cfg.Rico,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return ricoadapter.NewAdapter(ricoclient.New(opts...), now)
			}),
		},
		{
			Name: "Scapp", Variable: scappSwitchVariable, Config: cfg.Scapp,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return scappadapter.NewAdapter(scappclient.New(opts...), now)
			}),
		},
		{
			Name: "Kursi.ge", Variable: kursiSwitchVariable, Config: cfg.Kursi,
			Build: cached(func(opts ...vendorhttp.Option) app.CommercialRateSource {
				return kursiadapter.NewAdapter(kursiclient.New(opts...), now)
			}),
		},
	}
}

func cached(
	adapter func(opts ...vendorhttp.Option) app.CommercialRateSource,
) func(opts ...vendorhttp.Option) (*ratecache.CommercialRateCache, error) {
	return func(opts ...vendorhttp.Option) (*ratecache.CommercialRateCache, error) {
		return ratecache.NewCommercial(adapter(opts...))
	}
}

// Register adds the tools of the context. The commercial tool needs one bank or more.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if err := tool.Register(registry, m.getRates, m.history); err != nil {
		return err
	}

	if len(m.banks) == 0 {
		return nil
	}

	return tool.RegisterCommercial(registry, m.commercial)
}

// Close stops every cache.
func (m *Module) Close() error {
	return errors.Join(m.rateSets.Close(), vendors.CloseAll(m.banks))
}
