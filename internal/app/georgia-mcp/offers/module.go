// Package offers is the bounded context of the card offers: the discounts, the cashback,
// and the points that the banks of Georgia give to their customers at partners.
package offers

import (
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
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	"github.com/abgeo/georgia-mcp/pkg/bogoffers"
	libertyclient "github.com/abgeo/georgia-mcp/pkg/liberty"
	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
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

	registry := vendors.New[*offercache.OfferCache](logger)
	registry.Add(vendors.Entry[*offercache.OfferCache]{
		Name: "BOG", Enabled: cfg.BOG.Enabled, Variable: bogSwitchVariable,
		Build: func() (*offercache.OfferCache, error) { return bogSource(cfg.BOG, logger, now) },
	})
	registry.Add(vendors.Entry[*offercache.OfferCache]{
		Name: "TBC", Enabled: cfg.TBC.Enabled, Variable: tbcSwitchVariable,
		Build: func() (*offercache.OfferCache, error) { return tbcSource(cfg.TBC, logger, now) },
	})
	registry.Add(vendors.Entry[*offercache.OfferCache]{
		Name: "Liberty", Enabled: cfg.Liberty.Enabled, Variable: libertySwitchVariable,
		Build: func() (*offercache.OfferCache, error) {
			return libertySource(cfg.Liberty, logger, now)
		},
	})

	banks, err := registry.Build()
	if err != nil {
		return nil, fmt.Errorf("building the banks: %w", err)
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
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
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
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
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
		vendorhttp.WithBaseURL(cfg.BaseURL.String()),
		vendorhttp.WithHTTPClient(httpclient.New(httpclient.Config(cfg.HTTP))),
	)

	liberty, err := offercache.New(libertyadapter.NewAdapter(client, now, logger))
	if err != nil {
		return nil, fmt.Errorf("building the cache of Liberty: %w", err)
	}

	return liberty, nil
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
	return vendors.CloseAll(m.banks)
}
