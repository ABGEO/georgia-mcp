package permits

import (
	"fmt"
	"log/slog"
	"time"

	permitcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/cache"
	tasadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/tas"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	tasclient "github.com/abgeo/georgia-mcp/pkg/tas"
)

const switchVariable = "GEOMCP_PERMITS_TAS_ENABLED"

// Module holds the use cases of the context and their cache. A module of a TAS that is
// off holds none.
type Module struct {
	uses    *tool.UseCases
	sources *permitcache.SourceCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use cases, unless the operator
// turned TAS off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "permits"))

	if !cfg.TAS.Enabled {
		logger.Info("TAS is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	adapter := tasadapter.NewAdapter(
		tasclient.New(cfg.TAS.ClientOptions(nil, logger.With(slog.String("vendor", "TAS")))...),
		now,
	)

	sources, err := permitcache.NewSourceCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the permits: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{
			SearchApplications: app.NewSearchApplications(sources),
			GetApplication:     app.NewGetApplication(sources),
		},
		sources: sources,
	}, nil
}

// Register adds the tools of the context. A TAS that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the cache.
func (m *Module) Close() error {
	if m.uses == nil {
		return nil
	}

	return m.sources.Close()
}
