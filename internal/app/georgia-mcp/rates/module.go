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
	nbgadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/nbg"
	tbcadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tbc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	bogclient "github.com/abgeo/georgia-mcp/pkg/bog"
	nbgclient "github.com/abgeo/georgia-mcp/pkg/nbg"
	tbcclient "github.com/abgeo/georgia-mcp/pkg/tbc"
)

const (
	tbcKeyVariable    = "GEOMCP_RATES_TBC_API_KEY"
	bogSwitchVariable = "GEOMCP_RATES_BOG_ENABLED"
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

// commercialSources builds the cached source of each bank that the server can read.
func commercialSources(
	cfg Config,
	logger *slog.Logger,
	now func() time.Time,
) ([]*ratecache.CommercialRateCache, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "rates"))

	var banks []*ratecache.CommercialRateCache

	if cfg.TBC.APIKey == "" {
		logger.Warn("TBC Bank is off: no API key", slog.String("variable", tbcKeyVariable))
	} else {
		tbc, err := tbcSource(cfg.TBC, now)
		if err != nil {
			return nil, err
		}

		banks = append(banks, tbc)
	}

	if !cfg.BOG.Enabled {
		logger.Info("Bank of Georgia is off", slog.String("variable", bogSwitchVariable))

		return banks, nil
	}

	bog, err := bogSource(cfg.BOG, now)
	if err != nil {
		for _, bank := range banks {
			_ = bank.Close()
		}

		return nil, err
	}

	return append(banks, bog), nil
}

func tbcSource(cfg TBCConfig, now func() time.Time) (*ratecache.CommercialRateCache, error) {
	client := tbcclient.New(
		tbcclient.WithBaseURL(cfg.BaseURL.String()),
		tbcclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
		tbcclient.WithAPIKey(cfg.APIKey),
	)

	tbc, err := ratecache.NewCommercial(tbcadapter.NewAdapter(client, now))
	if err != nil {
		return nil, fmt.Errorf("building the cache of TBC Bank: %w", err)
	}

	return tbc, nil
}

func bogSource(cfg BOGConfig, now func() time.Time) (*ratecache.CommercialRateCache, error) {
	client := bogclient.New(
		bogclient.WithBaseURL(cfg.BaseURL.String()),
		bogclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	bog, err := ratecache.NewCommercial(bogadapter.NewAdapter(client, now))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Bank of Georgia: %w", err)
	}

	return bog, nil
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
