package holidays_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	testdata     = "../../../../pkg/nager/testdata/ge-2026.json"
	listHolidays = "holidays_list_holidays"
	getDay       = "holidays_get_day"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

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

// fakeNager serves the recorded answer of 2026, or the status, and counts the calls.
type fakeNager struct {
	server *httptest.Server
	calls  atomic.Int32
}

func newFakeNager(t *testing.T, status int) *fakeNager {
	t.Helper()

	body, err := os.ReadFile(testdata)
	if err != nil {
		t.Fatal(err)
	}

	fake := &fakeNager{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fake.calls.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakeNager) config(t *testing.T) holidays.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return holidays.Config{Nager: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 10},
	}}
}

func serve(t *testing.T, cfg holidays.Config, logger *slog.Logger) *mcp.ClientSession {
	t.Helper()

	module, err := holidays.NewWithClock(cfg, logger, clock)
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

func call(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	return result
}

// readOnly reports whether the annotations are those of MCP-004.
func readOnly(annotations *mcp.ToolAnnotations) bool {
	return annotations != nil && annotations.ReadOnlyHint && annotations.IdempotentHint &&
		annotations.DestructiveHint != nil && !*annotations.DestructiveHint &&
		annotations.OpenWorldHint != nil && *annotations.OpenWorldHint
}

// HOLIDAY-SC-019: With Nager.Date on, the server offers the two read-only tools.
func TestTheSwitchDecidesTheTools(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeNager(t, http.StatusOK).config(t), slog.New(slog.DiscardHandler))

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(tools.Tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(tools.Tools))
	}

	for _, item := range tools.Tools {
		if (item.Name != listHolidays && item.Name != getDay) || !readOnly(item.Annotations) {
			t.Errorf("tool %s, want a read-only tool of holidays", item.Name)
		}
	}
}

// HOLIDAY-SC-019: With Nager.Date off, the server offers no tool of holidays and says so.
func TestTheSwitchTurnsNagerOff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	off := serve(t, holidays.Config{}, slog.New(slog.NewJSONHandler(sink, nil)))

	none, err := off.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(none.Tools) != 0 || !strings.Contains(sink.String(), `"msg":"Nager.Date is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_HOLIDAYS_NAGER_ENABLED"`) {
		t.Errorf("tools = %d, log = %s, want no tool and the record Nager.Date is off",
			len(none.Tools), sink)
	}
}

// HOLIDAY-SC-015: A failing Nager.Date with an empty cache gives the message of the outage.
func TestAFailingNager(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeNager(t, http.StatusInternalServerError).config(t),
		slog.New(slog.DiscardHandler))

	want := "Nager.Date did not answer. Try again in a few minutes."

	for _, name := range []string{listHolidays, getDay} {
		result := call(t, session, name, map[string]any{})

		text, ok := result.Content[0].(*mcp.TextContent)
		if !result.IsError || !ok || text.Text != want {
			t.Errorf("%s = %+v, want the tool error %q", name, result.Content, want)
		}
	}
}

// HOLIDAY-SC-021: The list and a date, in two languages, send one call to Nager.Date.
func TestQuestionsShareOneRead(t *testing.T) {
	t.Parallel()

	fake := newFakeNager(t, http.StatusOK)
	session := serve(t, fake.config(t), slog.New(slog.DiscardHandler))

	for _, language := range []string{"en", "ka"} {
		for name, args := range map[string]map[string]any{
			listHolidays: {"language": language},
			getDay:       {"date": "2026-05-26", "language": language},
		} {
			if result := call(t, session, name, args); result.IsError {
				t.Fatalf("%s = %+v, want a result", name, result.Content)
			}
		}
	}

	if calls := fake.calls.Load(); calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}
