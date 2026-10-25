package cinema_test

import (
	"bytes"
	"context"
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

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	testdata      = "../../../../pkg/cavea/testdata"
	streetFighter = "6ab27d3fde5280679e341878"
	galleria2215  = "cag_5/27-10-2026/22:15"
	seatTicket    = "/api/booking/seat-ticket/" + galleria2215
	seats         = "/api/booking/seats/" + galleria2215
	listSessions  = "cinema_list_sessions"
	getSeatMap    = "cinema_get_seat_map"
	keySession    = "session"
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 14, 0, 0, 0, kernel.TbilisiZone())
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

// fakeCavea serves the recorded answers of Cavea and counts the calls of each path.
type fakeCavea struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  map[string]int
}

func newFakeCavea(t *testing.T) *fakeCavea {
	t.Helper()

	files := map[string]string{
		"/api/theaters":          "theaters.json",
		"/api/theaters/hall-map": "hall-map.json",
		"/movies":                "movies.rsc",
		"/movies?coming-soon":    "movies-coming-soon.rsc",
		"/api/movies/search":     "search-street.json",
		"/sessions":              "sessions.rsc",
		"/api/booking/sessions/all/" + streetFighter: "sessions-all-street-fighter.json",
		"/api/sessions/availability":                 "availability.json",
		seatTicket:                                   "seat-ticket.json",
		seats:                                        "seats-cag5.json",
	}

	fake := &fakeCavea{calls: map[string]int{}}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if r.URL.Query().Get("filter") == "coming-soon" {
			path += "?coming-soon"
		}

		fake.mu.Lock()
		fake.calls[path]++
		fake.mu.Unlock()

		name, ok := files[path]
		if !ok {
			http.NotFound(w, r)

			return
		}

		if path == seatTicket {
			w.Header().Add("Set-Cookie", "cavea_vid=visitor; Path=/")
		}

		body, err := os.ReadFile(
			filepath.Join(testdata, name),
		)
		if err != nil {
			t.Error(err)
		}

		_, _ = w.Write(body)
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakeCavea) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[path]
}

func (f *fakeCavea) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	sum := 0
	for _, calls := range f.calls {
		sum += calls
	}

	return sum
}

func (f *fakeCavea) config(t *testing.T) cinema.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return cinema.Config{Cavea: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 10},
	}}
}

func serve(t *testing.T, cfg cinema.Config, logger *slog.Logger) *mcp.ClientSession {
	t.Helper()

	module, err := cinema.NewWithClock(cfg, logger, readAt)
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

func call(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}

	if result.IsError {
		t.Fatalf("CallTool(%s) = %+v, want a result", name, result.Content)
	}
}

// readOnly reports whether the annotations are those of MCP-004.
func readOnly(annotations *mcp.ToolAnnotations) bool {
	return annotations != nil && annotations.ReadOnlyHint && annotations.IdempotentHint &&
		annotations.DestructiveHint != nil && !*annotations.DestructiveHint &&
		annotations.OpenWorldHint != nil && *annotations.OpenWorldHint
}

// CAVCINE-SC-024: With Cavea on, the server offers the five read-only tools. With Cavea
// off, it offers none and says so in the log.
func TestTheSwitchTurnsCaveaOff(t *testing.T) {
	t.Parallel()

	on, err := serve(
		t,
		newFakeCavea(t).config(t),
		slog.New(slog.DiscardHandler),
	).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(on.Tools))

	for _, listed := range on.Tools {
		if !readOnly(listed.Annotations) {
			t.Errorf(
				"annotations of %s = %+v, want those of MCP-004",
				listed.Name,
				listed.Annotations,
			)
		}

		names = append(names, listed.Name)
	}

	if len(names) != 5 {
		t.Errorf("tools = %v, want the five tools of cinema", names)
	}

	sink := &syncBuffer{}

	off, err := serve(
		t,
		cinema.Config{},
		slog.New(slog.NewJSONHandler(sink, nil)),
	).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(off.Tools) != 0 || !strings.Contains(sink.String(), `"msg":"Cavea is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_CINEMA_CAVEA_ENABLED"`) {
		t.Errorf(
			"tools = %d, log = %s, want no tool and the record Cavea is off",
			len(off.Tools),
			sink,
		)
	}
}

// CAVCINE-SC-022: The seats have no cache. Each seat map sends its 2 seat calls.
func TestEachSeatMapReadsTheSeats(t *testing.T) {
	t.Parallel()

	fake := newFakeCavea(t)
	session := serve(t, fake.config(t), slog.New(slog.DiscardHandler))

	call(t, session, getSeatMap, map[string]any{keySession: galleria2215})
	call(t, session, getSeatMap, map[string]any{keySession: galleria2215})

	if tickets, maps := fake.count(seatTicket), fake.count(seats); tickets != 2 || maps != 2 {
		t.Errorf("seat calls = %d tickets and %d maps, want 2 and 2", tickets, maps)
	}
}

// CAVCINE-SC-023: With the sessions in the cache, a seat map sends only the POST and the
// GET of the seats.
func TestASeatMapSendsTwoCalls(t *testing.T) {
	t.Parallel()

	fake := newFakeCavea(t)
	session := serve(t, fake.config(t), slog.New(slog.DiscardHandler))

	call(t, session, listSessions, map[string]any{"date": "2026-10-27"})
	call(t, session, listSessions, map[string]any{"movie": "street-fighter", "date": "2026-10-27"})

	before := fake.total()

	call(t, session, getSeatMap, map[string]any{keySession: galleria2215})

	if sent := fake.total() - before; sent != 2 || fake.count(seatTicket) != 1 ||
		fake.count(seats) != 1 {
		t.Errorf("seat map sent %d calls, want the POST and the GET of the seats", sent)
	}
}
