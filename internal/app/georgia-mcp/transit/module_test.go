package transit_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	testdata = "../../../../pkg/ttc/testdata"
	testKey  = "test-key"
	keyLat   = "lat"
	keyLon   = "lon"
	keyQuery = "query"
)

func readAt() time.Time {
	return time.Date(2026, 10, 19, 22, 10, 24, 0, kernel.TbilisiZone())
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

// fakeTTC serves the recorded answers of the TTC and keeps the instant of each call.
type fakeTTC struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  []string
	times  []time.Time
}

func files() map[string]string {
	return map[string]string{
		"/v2/stops en":                    "stops-en.json",
		"/v2/stops ka":                    "stops-ka.json",
		"/v3/routes en":                   "routes-en.json",
		"/v3/routes ka":                   "routes-ka.json",
		"/v2/stops/1:1000/arrival-times ": "arrival-times.json",
		"/v2/plan ":                       "plan.json",
	}
}

func newFakeTTC(t *testing.T) *fakeTTC {
	t.Helper()

	fake := &fakeTTC{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.calls = append(fake.calls, r.URL.Path)
		fake.times = append(fake.times, time.Now())
		fake.mu.Unlock()

		if r.Header.Get("X-Api-Key") != testKey {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		selector := r.URL.Query().Get("locale")
		if r.URL.Path != "/v2/stops" && r.URL.Path != "/v3/routes" {
			selector = r.URL.Query().Get("routeId")
		}

		name, found := files()[r.URL.Path+" "+selector]
		if !found {
			_, _ = w.Write([]byte("[]"))

			return
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

func (f *fakeTTC) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	count := 0

	for _, path := range f.calls {
		if strings.HasPrefix(path, prefix) {
			count++
		}
	}

	return count
}

func (f *fakeTTC) config(t *testing.T) transit.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return transit.Config{TTC: transit.TTCConfig{
		Enabled: true,
		APIKey:  testKey,
		BaseURL: *base,
		HTTP:    transit.TTCHTTPConfig{Timeout: 10 * time.Second, Rate: 4, Burst: 1},
	}}
}

func serve(t *testing.T, cfg transit.Config, logger *slog.Logger) *mcp.ClientSession {
	t.Helper()

	module, err := transit.NewWithClock(cfg, logger, readAt)
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

func call[T any](t *testing.T, session *mcp.ClientSession, name string, args map[string]any) T {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}

	if result.IsError {
		t.Fatalf("CallTool(%s) = %+v, want a result", name, result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out T
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

// TTCTRANS-SC-036: With the TTC on, the server offers the eight read-only tools.
func TestTheSwitchDecidesTheTools(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeTTC(t).config(t), slog.New(slog.DiscardHandler))

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(tools.Tools))

	for _, listed := range tools.Tools {
		names = append(names, listed.Name)

		if !readOnly(listed.Annotations) {
			t.Errorf(
				"%s has the annotations %+v, want those of MCP-004",
				listed.Name,
				listed.Annotations,
			)
		}
	}

	want := []string{
		"transit_get_route", "transit_get_timetable", "transit_list_arrivals",
		"transit_list_nearby_stops", "transit_plan_trip", "transit_search_places",
		"transit_search_routes", "transit_search_stops",
	}
	if diff := cmp.Diff(want, names); diff != "" {
		t.Errorf("tools mismatch (-want +got):\n%s", diff)
	}
}

// TTCTRANS-SC-036: With the TTC off, the server offers no tool of transport and says so
// in the log.
func TestTheSwitchTurnsTheTTCOff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	off := serve(t, transit.Config{}, slog.New(slog.NewJSONHandler(sink, nil)))

	none, err := off.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(none.Tools) != 0 || !strings.Contains(sink.String(), `"msg":"TTC is off"`) {
		t.Errorf(
			"tools = %d, log = %s, want no tool and the record TTC is off",
			len(none.Tools),
			sink,
		)
	}
}

type arrivalsOutput struct {
	Source   string `json:"source"`
	AsOf     string `json:"as_of"`
	Arrivals []struct {
		Route    string `json:"route"`
		Kind     string `json:"kind"`
		Minutes  int    `json:"minutes"`
		Expected string `json:"expected"`
		Live     bool   `json:"live"`
	} `json:"arrivals"`
}

// TTCTRANS-SC-037: The arrivals at a stop carry the source, the instant of the read in
// Tbilisi, and an expected instant of as_of plus the minutes.
func TestListArrivals(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeTTC(t).config(t), slog.New(slog.DiscardHandler))

	out := call[arrivalsOutput](t, session, "transit_list_arrivals", map[string]any{"stop": "1000"})

	if out.Source != "TTC" || out.AsOf != "2026-10-19T22:10:24+04:00" {
		t.Errorf(
			"source, as_of = %q, %q, want TTC and 2026-10-19T22:10:24+04:00",
			out.Source,
			out.AsOf,
		)
	}

	first := out.Arrivals[0]
	if first.Route != "531" || first.Kind != "minibus" || !first.Live || first.Minutes != 1 {
		t.Errorf("first arrival = %+v, want the minibus 531, live, in 1 minute", first)
	}

	asOf, _ := time.Parse(time.RFC3339, out.AsOf)

	for _, arrival := range out.Arrivals {
		want := asOf.Add(time.Duration(arrival.Minutes) * time.Minute).
			In(kernel.TbilisiZone()).
			Format(time.RFC3339)
		if arrival.Expected != want {
			t.Errorf("%s expected %s, want %s", arrival.Route, arrival.Expected, want)
		}
	}
}

type tripOutput struct {
	Itineraries []struct {
		DurationMinutes int `json:"duration_minutes"`
		Transfers       int `json:"transfers"`
		Legs            []struct {
			Kind        string `json:"kind"`
			RouteNumber string `json:"route_number"`
			Start       string `json:"start"`
			End         string `json:"end"`
			DistanceM   int    `json:"distance_m"`
			From        struct {
				StopID string `json:"stop_id"`
			} `json:"from"`
		} `json:"legs"`
	} `json:"itineraries"`
}

// TTCTRANS-SC-039: A trip plan gives 3 itineraries. The first takes 61 minutes with 1
// transfer, and rides the bus 310 from 21:42 to 21:59 over 4,913 m.
func TestPlanTrip(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeTTC(t).config(t), slog.New(slog.DiscardHandler))

	out := call[tripOutput](t, session, "transit_plan_trip", map[string]any{
		"from": map[string]any{keyLat: 41.7251, keyLon: 44.7554},
		"to":   map[string]any{keyLat: 41.7151, keyLon: 44.8271},
	})

	if len(out.Itineraries) != 3 {
		t.Fatalf("itineraries = %d, want 3", len(out.Itineraries))
	}

	first := out.Itineraries[0]
	if first.DurationMinutes != 61 || first.Transfers != 1 {
		t.Errorf(
			"first itinerary = %d minutes, %d transfers, want 61 and 1",
			first.DurationMinutes,
			first.Transfers,
		)
	}

	ride := first.Legs[1]
	if ride.Kind != "bus" || ride.RouteNumber != "310" || ride.DistanceM != 4913 ||
		ride.Start != "2026-10-19T21:42:51+04:00" || ride.End != "2026-10-19T21:59:48+04:00" ||
		ride.From.StopID != "1:1000" {
		t.Errorf("first ride = %+v, want the bus 310 from 1:1000, 21:42 to 21:59, 4913 m", ride)
	}
}

// TTCTRANS-SC-042: 30 searches of stops, of nearby stops, and of routes send 4 calls to
// the TTC: the stops and the routes, each in two languages.
func TestSearchesReadTheListsOnce(t *testing.T) {
	t.Parallel()

	fake := newFakeTTC(t)
	session := serve(t, fake.config(t), slog.New(slog.DiscardHandler))

	for range 10 {
		call[map[string]any](t, session, "transit_search_stops", map[string]any{keyQuery: "park"})
		call[map[string]any](
			t,
			session,
			"transit_list_nearby_stops",
			map[string]any{keyLat: 41.7251, keyLon: 44.7554},
		)
		call[map[string]any](t, session, "transit_search_routes", map[string]any{keyQuery: "385"})
	}

	if stops, routes := fake.count(
		"/v2/stops",
	), fake.count(
		"/v3/routes",
	); stops != 2 ||
		routes != 2 {
		t.Errorf("the TTC got %d calls of stops and %d of routes, want 2 and 2", stops, routes)
	}
}

// TTCTRANS-SC-043: 20 calls of arrivals at once send at most 5 calls to the TTC in any
// second, with the defaults of the configuration.
func TestTheLimitBoundsTheCalls(t *testing.T) {
	t.Parallel()

	fake := newFakeTTC(t)
	session := serve(t, fake.config(t), slog.New(slog.DiscardHandler))

	call[map[string]any](t, session, "transit_search_stops", map[string]any{keyQuery: "park"})
	call[map[string]any](t, session, "transit_search_routes", map[string]any{keyQuery: "385"})

	var wait sync.WaitGroup

	for i := range 20 {
		wait.Go(func() {
			_, _ = session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "transit_list_arrivals",
				Arguments: map[string]any{"stop": "1000", "route": "385", "limit": i + 1},
			})
		})
	}

	wait.Wait()

	fake.mu.Lock()
	defer fake.mu.Unlock()

	for i := 0; i+5 < len(fake.times); i++ {
		if gap := fake.times[i+5].Sub(fake.times[i]); gap < time.Second {
			t.Fatalf(
				"calls %d to %d came within %v, want 5 calls or fewer in any second",
				i,
				i+5,
				gap,
			)
		}
	}
}
