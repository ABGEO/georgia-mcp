// Package postal is the bounded context of the postal codes of Georgia: the postal code
// of Georgian Post for an address, a street, a town, or a village.
package postal

import (
	"fmt"
	"log/slog"
	"time"

	postalcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/adapter/cache"
	gpostadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/adapter/gpost"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	gpostclient "github.com/abgeo/georgia-mcp/pkg/gpost"
)

const switchVariable = "GEOMCP_POSTAL_GPOST_ENABLED"

// Module holds the use case of the context and its cache. A module of a Georgian Post
// that is off holds none.
type Module struct {
	uses      *tool.UseCases
	addresses *postalcache.AddressCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use case, unless the operator
// turned Georgian Post off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "postal"))

	if !cfg.GPost.Enabled {
		logger.Info("Georgian Post is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	client := gpostclient.New(cfg.GPost.ClientOptions(nil)...)

	addresses, err := postalcache.NewAddressCache(gpostadapter.NewAdapter(client, now))
	if err != nil {
		return nil, fmt.Errorf("building the cache of the addresses: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses:      &tool.UseCases{SearchCodes: app.NewSearchCodes(addresses)},
		addresses: addresses,
	}, nil
}

// Register adds the tool of the context. A Georgian Post that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the cache.
func (m *Module) Close() error {
	if m.addresses == nil {
		return nil
	}

	return m.addresses.Close()
}
