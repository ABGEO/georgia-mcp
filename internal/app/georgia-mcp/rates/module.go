// Package rates is the bounded context of the exchange rates: the official rates of
// the National Bank of Georgia, and the commercial rates of the banks.
package rates

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	bogadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/bog"
	ratecache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/cache"
	kursiadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/kursi"
	libertyadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/liberty"
	nbgadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/nbg"
	ricoadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/rico"
	tbcadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tbc"
	terabankadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/terabank"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	bogclient "github.com/abgeo/georgia-mcp/pkg/bog"
	kursiclient "github.com/abgeo/georgia-mcp/pkg/kursi"
	libertyclient "github.com/abgeo/georgia-mcp/pkg/libertyrates"
	nbgclient "github.com/abgeo/georgia-mcp/pkg/nbg"
	ricoclient "github.com/abgeo/georgia-mcp/pkg/rico"
	tbcclient "github.com/abgeo/georgia-mcp/pkg/tbc"
	terabankclient "github.com/abgeo/georgia-mcp/pkg/terabank"
)

const (
	tbcKeyVariable         = "GEOMCP_RATES_TBC_API_KEY"
	kursiSwitchVariable    = "GEOMCP_RATES_KURSI_ENABLED"
	bogSwitchVariable      = "GEOMCP_RATES_BOG_ENABLED"
	libertySwitchVariable  = "GEOMCP_RATES_LIBERTY_ENABLED"
	terabankSwitchVariable = "GEOMCP_RATES_TERABANK_ENABLED"
	ricoSwitchVariable     = "GEOMCP_RATES_RICO_ENABLED"
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

// New builds the clients, the adapters, the caches, and the use cases. A bank with no
// key is left out, with a warning. A bank that the operator turned off is left out.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	client := nbgclient.New(
		nbgclient.WithBaseURL(cfg.NBG.BaseURL.String()),
		nbgclient.WithHTTPClient(httpclient.New(cfg.NBG.HTTP)),
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

// switchedBank is a bank that needs no key, so the operator turns it off with a switch.
type switchedBank struct {
	enabled  bool
	name     string
	variable string
	adapter  func() app.CommercialRateSource
}

// commercialSources builds the cached source of each bank that the server can read.
func commercialSources(
	cfg Config,
	logger *slog.Logger,
	now func() time.Time,
) ([]*ratecache.CommercialRateCache, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "rates"))

	var adapters []app.CommercialRateSource

	if cfg.TBC.APIKey == "" {
		logger.Warn("TBC Bank is off: no API key", slog.String("variable", tbcKeyVariable))
	} else {
		adapters = append(adapters, tbcAdapter(cfg.TBC, now))
	}

	for _, bank := range switchedBanks(cfg, now) {
		if !bank.enabled {
			logger.Info(bank.name+" is off", slog.String("variable", bank.variable))

			continue
		}

		adapters = append(adapters, bank.adapter())
	}

	banks := make([]*ratecache.CommercialRateCache, 0, len(adapters))

	for _, adapter := range adapters {
		bank, err := ratecache.NewCommercial(adapter)
		if err != nil {
			for _, built := range banks {
				_ = built.Close()
			}

			return nil, fmt.Errorf("building the cache of %s: %w", adapter.Bank().Name, err)
		}

		banks = append(banks, bank)
	}

	return banks, nil
}

func switchedBanks(cfg Config, now func() time.Time) []switchedBank {
	return []switchedBank{
		{
			cfg.BOG.Enabled,
			"Bank of Georgia",
			bogSwitchVariable,
			func() app.CommercialRateSource { return bogAdapter(cfg.BOG, now) },
		},
		{
			cfg.Liberty.Enabled,
			"Liberty Bank",
			libertySwitchVariable,
			func() app.CommercialRateSource { return libertyAdapter(cfg.Liberty, now) },
		},
		{
			cfg.Terabank.Enabled,
			"Terabank",
			terabankSwitchVariable,
			func() app.CommercialRateSource { return terabankAdapter(cfg.Terabank, now) },
		},
		{
			cfg.Rico.Enabled,
			"Rico Credit",
			ricoSwitchVariable,
			func() app.CommercialRateSource { return ricoAdapter(cfg.Rico, now) },
		},
		{
			cfg.Kursi.Enabled,
			"Kursi.ge",
			kursiSwitchVariable,
			func() app.CommercialRateSource { return kursiAdapter(cfg.Kursi, now) },
		},
	}
}

func bogAdapter(cfg BOGConfig, now func() time.Time) *bogadapter.Adapter {
	return bogadapter.NewAdapter(bogclient.New(
		bogclient.WithBaseURL(cfg.BaseURL.String()),
		bogclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	), now)
}

func libertyAdapter(cfg LibertyConfig, now func() time.Time) *libertyadapter.Adapter {
	return libertyadapter.NewAdapter(libertyclient.New(
		libertyclient.WithBaseURL(cfg.BaseURL.String()),
		libertyclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	), now)
}

func terabankAdapter(cfg TerabankConfig, now func() time.Time) *terabankadapter.Adapter {
	return terabankadapter.NewAdapter(terabankclient.New(
		terabankclient.WithBaseURL(cfg.BaseURL.String()),
		terabankclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	), now)
}

func ricoAdapter(cfg RicoConfig, now func() time.Time) *ricoadapter.Adapter {
	return ricoadapter.NewAdapter(ricoclient.New(
		ricoclient.WithBaseURL(cfg.BaseURL.String()),
		ricoclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	), now)
}

func kursiAdapter(cfg KursiConfig, now func() time.Time) *kursiadapter.Adapter {
	return kursiadapter.NewAdapter(kursiclient.New(
		kursiclient.WithBaseURL(cfg.BaseURL.String()),
		kursiclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	), now)
}

func tbcAdapter(cfg TBCConfig, now func() time.Time) *tbcadapter.Adapter {
	return tbcadapter.NewAdapter(tbcclient.New(
		tbcclient.WithBaseURL(cfg.BaseURL.String()),
		tbcclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
		tbcclient.WithAPIKey(cfg.APIKey),
	), now)
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
	errs := make([]error, 0, 1+len(m.banks))
	errs = append(errs, m.rateSets.Close())

	for _, bank := range m.banks {
		errs = append(errs, bank.Close())
	}

	return errors.Join(errs...)
}
