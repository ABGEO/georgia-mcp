package postal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	testdata    = "../../../../pkg/gpost/testdata"
	searchCodes = "postal_search_codes"
	keyQuery    = "query"
	rustavi     = "რუსთავი"
)

func readAt() time.Time {
	return time.Date(2026, 10, 24, 12, 0, 0, 0, kernel.TbilisiZone())
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

// fakePost serves the recorded page and the recorded answer for რუსთავი in English. Any
// other search gives the recorded answer with no match. It counts each call.
type fakePost struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  int
}

func newFakePost(t *testing.T) *fakePost {
	t.Helper()

	fake := &fakePost{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.calls++
		fake.mu.Unlock()

		name := "find-none-ka.html"

		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Set-Cookie", "__RequestVerificationToken=test-token; path=/; HttpOnly")

			name = "postal-codes.html"
		case r.FormValue("postalCodeSearchValue") == rustavi &&
			strings.HasSuffix(r.Header.Get("Cookie"), "lang=en"):
			name = "find-rustavi-en.html"
		}

		path := filepath.Join(testdata, name)

		body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
		if err != nil {
			t.Error(err)
		}

		_, _ = w.Write(body)
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakePost) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls
}

func (f *fakePost) config(t *testing.T) postal.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return postal.Config{GPost: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 10},
	}}
}

func serve(t *testing.T, cfg postal.Config, logger *slog.Logger) *mcp.ClientSession {
	t.Helper()

	module, err := postal.NewWithClock(cfg, logger, readAt)
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

type codesOutput struct {
	Source    string          `json:"source"`
	AsOf      string          `json:"as_of"`
	Total     int             `json:"total"`
	Addresses []addressOutput `json:"addresses"`
}

type addressOutput struct {
	PostalCode string `json:"postal_code"`
	Address    string `json:"address"`
	District   string `json:"district"`
}

func search(t *testing.T, session *mcp.ClientSession, args map[string]any) codesOutput {
	t.Helper()

	result, err := session.CallTool(
		t.Context(),
		&mcp.CallToolParams{Name: searchCodes, Arguments: args},
	)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if result.IsError {
		t.Fatalf("CallTool() = %+v, want a result", result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out codesOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

// readOnly reports whether the annotations are those of MCP-004.
func readOnly(annotations *mcp.ToolAnnotations) bool {
	return annotations != nil && annotations.ReadOnlyHint && annotations.IdempotentHint &&
		annotations.DestructiveHint != nil && !*annotations.DestructiveHint &&
		annotations.OpenWorldHint != nil && *annotations.OpenWorldHint
}

// POSTCODE-SC-020: With Georgian Post on, the server offers the read-only tool of the
// postal codes.
func TestTheSwitchDecidesTheTool(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakePost(t).config(t), slog.New(slog.DiscardHandler))

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(tools.Tools) != 1 || tools.Tools[0].Name != searchCodes ||
		!readOnly(tools.Tools[0].Annotations) {
		t.Errorf("tools = %+v, want the read-only tool %s", tools.Tools, searchCodes)
	}
}

// POSTCODE-SC-020: With Georgian Post off, the server offers no tool of postal codes and
// says so in the log.
func TestTheSwitchTurnsGeorgianPostOff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	off := serve(t, postal.Config{}, slog.New(slog.NewJSONHandler(sink, nil)))

	none, err := off.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(none.Tools) != 0 || !strings.Contains(sink.String(), `"msg":"Georgian Post is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_POSTAL_GPOST_ENABLED"`) {
		t.Errorf("tools = %d, log = %s, want no tool and the record Georgian Post is off",
			len(none.Tools), sink)
	}
}

// POSTCODE-SC-014: The tool reads the recorded answer of Georgian Post through the whole
// context.
func TestSearchCodesReadsGeorgianPost(t *testing.T) {
	t.Parallel()

	out := search(t, serve(t, newFakePost(t).config(t), slog.New(slog.DiscardHandler)),
		map[string]any{keyQuery: rustavi, "limit": 2})

	want := codesOutput{
		Source: "Georgian Post",
		AsOf:   "2026-10-24T12:00:00+04:00",
		Total:  12,
		Addresses: []addressOutput{
			{PostalCode: "3700", Address: "City Rustavi", District: "Rustavi City"},
			{PostalCode: "0500", Address: "Village Rustavi", District: "Aspindzis District"},
		},
	}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// POSTCODE-SC-018: A second search of the same text in the same language sends no call
// to Georgian Post. A search in another language sends its own 2 calls.
func TestSearchesShareTheCache(t *testing.T) {
	t.Parallel()

	fake := newFakePost(t)
	session := serve(t, fake.config(t), slog.New(slog.DiscardHandler))

	georgian := map[string]any{keyQuery: rustavi, "language": "ka"}

	search(t, session, georgian)
	search(t, session, georgian)

	if calls := fake.count(); calls != 2 {
		t.Errorf("calls after two searches in ka = %d, want 2", calls)
	}

	search(t, session, map[string]any{keyQuery: rustavi, "language": "en"})

	if calls := fake.count(); calls != 4 {
		t.Errorf("calls after a search in en = %d, want 4", calls)
	}
}
