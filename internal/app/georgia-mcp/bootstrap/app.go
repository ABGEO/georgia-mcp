package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/logger"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// App is the process with every context registered.
type App struct {
	cfg      Config
	logger   *slog.Logger
	registry *mcpserver.Registry
	modules  []mcpserver.Module
}

// New builds the logger, each context, and the MCP server with every tool.
func New(cfg Config, logOutput io.Writer) (*App, error) {
	info := buildinfo.Get()
	log := logger.New(cfg.Log, logOutput, info)

	contexts, err := modules(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("building the contexts: %w", err)
	}

	app := &App{
		cfg:      cfg,
		logger:   log,
		registry: mcpserver.New(info, log),
		modules:  contexts,
	}

	if err := app.registry.Register(contexts...); err != nil {
		return nil, errors.Join(err, app.Close())
	}

	return app, nil
}

// Close closes each context that holds a resource. A failure to close one context
// is logged and does not stop the next.
func (a *App) Close() error {
	var errs []error

	for _, module := range a.modules {
		closer, ok := module.(io.Closer)
		if !ok {
			continue
		}

		if err := closer.Close(); err != nil {
			a.logger.Error("closing a context", slog.Any("error", err))
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// RunHTTP serves the tools over streamable HTTP until the context is cancelled.
func (a *App) RunHTTP(ctx context.Context) error {
	srv := httpserver.New(a.cfg.HTTP, a.logger, a.registry.Handler())
	srv.SetReady(true)

	a.logger.InfoContext(ctx, "serving http", slog.String("address", a.cfg.HTTP.Address))

	return srv.Run(ctx)
}

// RunStdio serves the tools over stdio until the input ends or the context is cancelled.
func (a *App) RunStdio(ctx context.Context) error {
	a.logger.InfoContext(ctx, "serving stdio")

	return a.registry.RunStdio(ctx, &mcp.StdioTransport{})
}
