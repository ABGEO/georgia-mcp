package parliament

import (
	"fmt"
	"log/slog"
	"time"

	parliamentcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/cache"
	parliamentadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/parliament"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	parliamentclient "github.com/abgeo/georgia-mcp/pkg/parliament"
)

const switchVariable = "GEOMCP_PARLIAMENT_PARLIAMENT_ENABLED"

// Module holds the use cases of the context and their cache. A module of a Parliament that
// is off holds none.
type Module struct {
	uses    *tool.UseCases
	sources *parliamentcache.SourceCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use cases, unless the operator
// turned the Parliament off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(
		slog.String("component", "context"),
		slog.String("context", "parliament"),
	)

	if !cfg.Parliament.Enabled {
		logger.Info("the Parliament is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	client := parliamentclient.New(
		cfg.Parliament.ClientOptions(nil, logger.With(slog.String("vendor", "Parliament")))...,
	)

	sources, err := parliamentcache.NewSourceCache(parliamentadapter.NewAdapter(client, now))
	if err != nil {
		return nil, fmt.Errorf("building the cache of the parliament: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{
			SearchBills:     app.NewSearchBills(sources),
			GetBill:         app.NewGetBill(sources),
			ListVotes:       app.NewListVotes(sources),
			GetVote:         app.NewGetVote(sources),
			SearchQuestions: app.NewSearchQuestions(sources, now),
		},
		sources: sources,
	}, nil
}

// Register adds the tools of the context. A Parliament that is off adds none.
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
