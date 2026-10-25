package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
)

const maxRequestBodyBytes = 1 << 20

// ErrDuplicateTool tells that two tools share one name.
var ErrDuplicateTool = errors.New("mcp server: duplicate tool name")

// Module is a bounded context that adds its tools to the registry.
type Module interface {
	Register(registry *Registry) error
}

// Registry holds the MCP server and the names of its tools. The process fills it
// at the start, from one goroutine.
type Registry struct {
	server *mcp.Server
	names  map[string]struct{}
}

// New returns a registry with no tool. The server declares the tools capability
// even with no tool, so a client gets an empty list and not an error.
func New(info buildinfo.Info, logger *slog.Logger) *Registry {
	server := mcp.NewServer(
		&mcp.Implementation{Name: buildinfo.Name, Version: info.Version},
		&mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}}},
	)

	server.AddReceivingMiddleware(logToolCalls(logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "tool-call"),
	)))

	return &Registry{server: server, names: map[string]struct{}{}}
}

// Register adds the tools of each module.
func (r *Registry) Register(modules ...Module) error {
	for _, module := range modules {
		if err := module.Register(r); err != nil {
			return fmt.Errorf("registering the tools: %w", err)
		}
	}

	return nil
}

// AddTool adds a read-only tool to the server. It returns ErrDuplicateTool when
// the name exists, because the server itself replaces such a tool with no error.
func AddTool[In, Out any](
	registry *Registry,
	tool *mcp.Tool,
	handler mcp.ToolHandlerFor[In, Out],
) error {
	if _, exists := registry.names[tool.Name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateTool, tool.Name)
	}

	registry.names[tool.Name] = struct{}{}

	annotated := *tool
	annotated.Annotations = readOnlyAnnotations()

	mcp.AddTool(registry.server, &annotated, handler)

	return nil
}

func readOnlyAnnotations() *mcp.ToolAnnotations {
	destructive, openWorld := false, true

	return &mcp.ToolAnnotations{
		ReadOnlyHint:    true,
		IdempotentHint:  true,
		DestructiveHint: &destructive,
		OpenWorldHint:   &openWorld,
	}
}

// Handler returns the streamable HTTP handler of the server. It keeps no session
// and answers JSON.
func (r *Registry) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return r.server },
		&mcp.StreamableHTTPOptions{
			Stateless:           true,
			JSONResponse:        true,
			MaxRequestBodyBytes: maxRequestBodyBytes,
		},
	)
}

// RunStdio serves one session over the transport until the input ends or the
// context is cancelled. Both are a clean stop.
func (r *Registry) RunStdio(ctx context.Context, transport mcp.Transport) error {
	err := r.server.Run(ctx, transport)
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) ||
		errors.Is(err, mcp.ErrConnectionClosed) {
		return nil
	}

	return fmt.Errorf("serving stdio: %w", err)
}
