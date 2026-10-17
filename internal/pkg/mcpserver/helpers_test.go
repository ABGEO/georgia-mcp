package mcpserver_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	echoTool    = "test_get_echo"
	failTool    = "test_get_failure"
	testVersion = "v0.1.0"
)

var errTestCause = errors.New("cause")

type echoInput struct {
	Text string `json:"text" jsonschema:"the text to return"`
}

type echoOutput struct {
	Text string `json:"text" jsonschema:"the text of the input"`
}

type echoModule struct{}

func (echoModule) Register(registry *mcpserver.Registry) error {
	return mcpserver.AddTool(
		registry,
		&mcp.Tool{Name: echoTool, Description: "Returns its input."},
		echo,
	)
}

func echo(
	_ context.Context,
	_ *mcp.CallToolRequest,
	in echoInput,
) (*mcp.CallToolResult, echoOutput, error) {
	return nil, echoOutput(in), nil
}

type failModule struct{}

func (failModule) Register(registry *mcpserver.Registry) error {
	return mcpserver.AddTool(registry, &mcp.Tool{Name: failTool, Description: "Fails."}, fail)
}

func fail(
	context.Context,
	*mcp.CallToolRequest,
	echoInput,
) (*mcp.CallToolResult, echoOutput, error) {
	return nil, echoOutput{}, &mcpserver.ToolError{Message: "the test failed", Cause: errTestCause}
}

func newRegistry(
	t *testing.T,
	logger *slog.Logger,
	modules ...mcpserver.Module,
) *mcpserver.Registry {
	t.Helper()

	registry := mcpserver.New(newInfo(), logger)
	if err := registry.Register(modules...); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	return registry
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newClient() *mcp.Client {
	return mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
}

func connectHTTP(t *testing.T, registry *mcpserver.Registry) *mcp.ClientSession {
	t.Helper()

	server := httptest.NewServer(registry.Handler())
	t.Cleanup(server.Close)

	transport := &mcp.StreamableClientTransport{
		Endpoint:             server.URL,
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}

	session, err := newClient().Connect(t.Context(), transport, nil)
	if err != nil {
		t.Fatalf("Connect() over HTTP error = %v", err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

func connectStdio(t *testing.T, registry *mcpserver.Registry) *mcp.ClientSession {
	t.Helper()

	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- registry.RunStdio(ctx, &mcp.IOTransport{Reader: serverReader, Writer: serverWriter})
	}()

	session, err := newClient().Connect(t.Context(), &mcp.IOTransport{Reader: clientReader, Writer: clientWriter}, nil)
	if err != nil {
		cancel()
		t.Fatalf("Connect() over stdio error = %v", err)
	}

	t.Cleanup(func() {
		_ = session.Close()

		cancel()

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("RunStdio() error = %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("RunStdio did not return")
		}
	})

	return session
}

func connectInMemory(t *testing.T, registry *mcpserver.Registry) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = registry.RunStdio(ctx, serverTransport) }()

	session, err := newClient().Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() in memory error = %v", err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()

	result, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}

	return names
}
