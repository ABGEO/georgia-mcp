package parliament_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

// syncBuffer is a log sink that the server and the test share.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p) //nolint:wrapcheck // A bytes.Buffer never fails.
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func enabled(t *testing.T) parliament.Config {
	t.Helper()

	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return parliament.Config{Parliament: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 10},
	}}
}

func serve(t *testing.T, cfg parliament.Config, logger *slog.Logger) *mcp.ClientSession {
	t.Helper()

	module, err := parliament.New(cfg, logger)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = module.Close() })

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, logger)
	if err := registry.Register(module); err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = registry.RunStdio(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).
		Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

// readOnly reports whether the annotations are those of MCP-004.
func readOnly(annotations *mcp.ToolAnnotations) bool {
	return annotations != nil && annotations.ReadOnlyHint && annotations.IdempotentHint &&
		annotations.DestructiveHint != nil && !*annotations.DestructiveHint &&
		annotations.OpenWorldHint != nil && *annotations.OpenWorldHint
}

// PARL-SC-021: With the Parliament on, the server offers the five read-only tools.
func TestTheSwitchDecidesTheTools(t *testing.T) {
	t.Parallel()

	tools, err := serve(t, enabled(t), slog.New(slog.DiscardHandler)).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(tools.Tools))

	for _, item := range tools.Tools {
		names = append(names, item.Name)

		if !readOnly(item.Annotations) {
			t.Errorf("tool %s is not read-only", item.Name)
		}
	}

	slices.Sort(names)

	want := []string{
		"parliament_get_bill",
		"parliament_get_vote",
		"parliament_list_votes",
		"parliament_search_bills",
		"parliament_search_questions",
	}
	if !slices.Equal(names, want) {
		t.Errorf("tools = %v, want %v", names, want)
	}
}

// PARL-SC-021: With the Parliament off, the server offers no tool of the parliament and
// says so.
func TestTheSwitchTurnsTheParliamentOff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	off := serve(t, parliament.Config{}, slog.New(slog.NewJSONHandler(sink, nil)))

	none, err := off.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(none.Tools) != 0 || !strings.Contains(sink.String(), `"msg":"the Parliament is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_PARLIAMENT_PARLIAMENT_ENABLED"`) {
		t.Errorf("tools = %d, log = %s, want no tool and the record of the switch",
			len(none.Tools), sink)
	}
}
