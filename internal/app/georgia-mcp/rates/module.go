// Package rates is the bounded context of the exchange rates. Its first source is the
// official rate of the National Bank of Georgia.
package rates

import (
	"fmt"
	"log/slog"
	"time"

	ratecache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/cache"
	nbgadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/nbg"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	nbgclient "github.com/abgeo/georgia-mcp/pkg/nbg"
)

// Module holds the use cases of the context and the cache they share.
type Module struct {
	getRates *app.GetOfficialRates
	history  *app.ListOfficialHistory
	rateSets *ratecache.RateSetCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the NBG client, the adapter, the cache, and the use cases.
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

	logger.Debug("built the context",
		slog.String("component", "context"),
		slog.String("context", "rates"),
		slog.String("nbg", cfg.NBG.BaseURL.String()),
	)

	return &Module{
		getRates: app.NewGetOfficialRates(rateSets, now),
		history:  app.NewListOfficialHistory(rateSets, now),
		rateSets: rateSets,
	}, nil
}

// Register adds the tools of the context.
func (m *Module) Register(registry *mcpserver.Registry) error {
	return tool.Register(registry, m.getRates, m.history)
}

// Close stops the cache.
func (m *Module) Close() error {
	return m.rateSets.Close()
}
