// Package offers is the bounded context of the card offers: the discounts, the cashback,
// and the points that the banks of Georgia give to their customers at partners.
package offers

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	bogadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/bog"
	offercache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/cache"
	libertyadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/liberty"
	tbcadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/tbc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/pkg/bogoffers"
	libertyclient "github.com/abgeo/georgia-mcp/pkg/liberty"
	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
)

const (
	bogSwitchVariable     = "GEOMCP_OFFERS_BOG_ENABLED"
	tbcSwitchVariable     = "GEOMCP_OFFERS_TBC_ENABLED"
	libertySwitchVariable = "GEOMCP_OFFERS_LIBERTY_ENABLED"
)

// Module holds the use cases of the context and the cache of each bank.
type Module struct {
	search *app.SearchOffers
	get    *app.GetOffer
	banks  []*offercache.OfferCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the clients, the adapters, the caches, and the use cases. A bank that the
// operator turned off is left out.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "offers"))

	var banks []*offercache.OfferCache

	if cfg.BOG.Enabled {
		bog, err := bogSource(cfg.BOG, logger, now)
		if err != nil {
			return nil, err
		}

		banks = append(banks, bog)
	} else {
		logger.Info("BOG is off", slog.String("variable", bogSwitchVariable))
	}

	if cfg.TBC.Enabled {
		tbc, err := tbcSource(cfg.TBC, logger, now)
		if err != nil {
			return nil, closeAll(banks, err)
		}

		banks = append(banks, tbc)
	} else {
		logger.Info("TBC is off", slog.String("variable", tbcSwitchVariable))
	}

	if cfg.Liberty.Enabled {
		liberty, err := libertySource(cfg.Liberty, logger, now)
		if err != nil {
			return nil, closeAll(banks, err)
		}

		banks = append(banks, liberty)
	} else {
		logger.Info("Liberty is off", slog.String("variable", libertySwitchVariable))
	}

	sources := make([]app.OfferSource, 0, len(banks))
	for _, bank := range banks {
		sources = append(sources, bank)
	}

	logger.Debug("built the context", slog.Int("banks", len(banks)))

	return &Module{
		search: app.NewSearchOffers(sources, now),
		get:    app.NewGetOffer(sources, now),
		banks:  banks,
	}, nil
}

func bogSource(
	cfg BOGConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*offercache.OfferCache, error) {
	client := bogoffers.New(
		bogoffers.WithBaseURL(cfg.BaseURL.String()),
		bogoffers.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	bog, err := offercache.New(bogadapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of BOG: %w", err)
	}

	return bog, nil
}

func tbcSource(
	cfg TBCConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*offercache.OfferCache, error) {
	client := tbcoffers.New(
		tbcoffers.WithBaseURL(cfg.BaseURL.String()),
		tbcoffers.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	tbc, err := offercache.New(tbcadapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of TBC: %w", err)
	}

	return tbc, nil
}

func libertySource(
	cfg LibertyConfig,
	logger *slog.Logger,
	now func() time.Time,
) (*offercache.OfferCache, error) {
	client := libertyclient.New(
		libertyclient.WithBaseURL(cfg.BaseURL.String()),
		libertyclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	liberty, err := offercache.New(libertyadapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Liberty: %w", err)
	}

	return liberty, nil
}

// closeAll closes the caches built before a failure, and returns the failure.
func closeAll(banks []*offercache.OfferCache, cause error) error {
	errs := make([]error, 0, 1+len(banks))
	errs = append(errs, cause)

	for _, bank := range banks {
		errs = append(errs, bank.Close())
	}

	return errors.Join(errs...)
}

// Register adds the tools of the context. The tools need one bank or more.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if len(m.banks) == 0 {
		return nil
	}

	return tool.Register(registry, m.search, m.get)
}

// Close stops every cache.
func (m *Module) Close() error {
	errs := make([]error, 0, len(m.banks))
	for _, bank := range m.banks {
		errs = append(errs, bank.Close())
	}

	return errors.Join(errs...)
}
