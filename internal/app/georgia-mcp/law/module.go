package law

import (
	"fmt"
	"log/slog"
	"time"

	lawcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/adapter/cache"
	matsneadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/adapter/matsne"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	matsneclient "github.com/abgeo/georgia-mcp/pkg/matsne"
)

const switchVariable = "GEOMCP_LAW_MATSNE_ENABLED"

// Module holds the use cases of the context and their cache. A module of a Legislative
// Herald that is off holds none.
type Module struct {
	uses *tool.UseCases
	acts *lawcache.ActCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use cases, unless the operator
// turned the Legislative Herald off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "law"))

	if !cfg.Matsne.Enabled {
		logger.Info("the Legislative Herald is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	vendorLogger := logger.With(slog.String("vendor", "Legislative Herald"))
	client := matsneclient.New(
		cfg.Matsne.ClientOptions(matsneclient.NewTransport(), vendorLogger)...,
	)

	acts, err := lawcache.NewActCache(matsneadapter.NewAdapter(client, now))
	if err != nil {
		return nil, fmt.Errorf("building the cache of the acts: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{
			SearchActs: app.NewSearchActs(acts, vendorLogger),
			GetAct:     app.NewGetAct(acts, vendorLogger),
		},
		acts: acts,
	}, nil
}

// Register adds the tools of the context. A Legislative Herald that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the cache.
func (m *Module) Close() error {
	if m.acts == nil {
		return nil
	}

	return m.acts.Close()
}
