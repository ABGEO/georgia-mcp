package air_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	testdata    = "../../../../pkg/nea/testdata/"
	getQuality  = "air_get_quality"
	listHistory = "air_list_history"
	keyCity     = "city"
	keyStation  = "station"
	keyLanguage = "language"
	tbilisi     = "Tbilisi"
	tsrt        = "TSRT"
)

func clock() time.Time {
	return time.Date(2026, time.November, 5, 0, 28, 0, 0, kernel.TbilisiZone())
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

// fakeNEA serves the recorded answers, or the status, and counts the calls of each path.
type fakeNEA struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  map[string]int
}

func newFakeNEA(t *testing.T, status int) *fakeNEA {
	t.Helper()

	read := func(name string) []byte {
		body, err := os.ReadFile(testdata + name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		return body
	}

	stations, municipalities := read("hourly-all.json"), read("municipalities.json")

	fake := &fakeNEA{calls: map[string]int{}}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.calls[r.URL.Path]++
		fake.mu.Unlock()

		w.WriteHeader(status)

		if strings.Contains(r.URL.Path, "municipalities") {
			_, _ = w.Write(municipalities)

			return
		}

		_, _ = w.Write(stations)
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakeNEA) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[path]
}

func (f *fakeNEA) config(t *testing.T) air.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return air.Config{NEA: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 10},
	}}
}

func serve(t *testing.T, cfg air.Config, logger *slog.Logger) *mcp.ClientSession {
	t.Helper()

	module, err := air.NewWithClock(cfg, logger, clock)
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

type readingOutput struct {
	Pollutant string `json:"pollutant"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
	Time      string `json:"time"`
}

type stationOutput struct {
	Code     string          `json:"code"`
	Address  string          `json:"address"`
	City     string          `json:"city"`
	Level    string          `json:"level"`
	Readings []readingOutput `json:"readings"`
}

type qualityOutput struct {
	Source   string          `json:"source"`
	AsOf     string          `json:"as_of"`
	Total    int             `json:"total"`
	Stations []stationOutput `json:"stations"`
}

func quality(t *testing.T, session *mcp.ClientSession, args map[string]any) qualityOutput {
	t.Helper()

	result := call(t, session, getQuality, args)
	if result.IsError {
		t.Fatalf("CallTool() = %+v, want a result", result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out qualityOutput
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

// NEAAIR-SC-024: With the NEA on, the server offers the two read-only tools.
func TestTheSwitchDecidesTheTools(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeNEA(t, http.StatusOK).config(t), slog.New(slog.DiscardHandler))

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(tools.Tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(tools.Tools))
	}

	for _, item := range tools.Tools {
		if (item.Name != getQuality && item.Name != listHistory) || !readOnly(item.Annotations) {
			t.Errorf("tool %s, want a read-only tool of air", item.Name)
		}
	}
}

// NEAAIR-SC-024: With the NEA off, the server offers no tool of air and says so.
func TestTheSwitchTurnsTheNEAOff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	off := serve(t, air.Config{}, slog.New(slog.NewJSONHandler(sink, nil)))

	none, err := off.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(none.Tools) != 0 || !strings.Contains(sink.String(), `"msg":"the NEA is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_AIR_NEA_ENABLED"`) {
		t.Errorf("tools = %d, log = %s, want no tool and the record the NEA is off",
			len(none.Tools), sink)
	}
}

// NEAAIR-SC-017: The quality of Tbilisi reads the recorded answer through the whole
// context.
func TestGetQualityReadsTheNEA(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeNEA(t, http.StatusOK).config(t), slog.New(slog.DiscardHandler))

	out := quality(t, session, map[string]any{keyCity: tbilisi})

	if out.Source != "National Environmental Agency" || out.AsOf != "2026-11-05T00:28:00+04:00" ||
		out.Total != 5 {
		t.Fatalf("head = %q, %q, %d", out.Source, out.AsOf, out.Total)
	}

	index := slices.IndexFunc(out.Stations, func(s stationOutput) bool { return s.Code == tsrt })
	if index < 0 {
		t.Fatal("no station TSRT")
	}

	station := out.Stations[index]
	first := station.Readings[0]
	station.Readings = nil

	want := stationOutput{
		Code: tsrt, Address: "Tbilisi - Akaki Tsereteli ave. №105", City: tbilisi, Level: "poor",
	}
	if diff := cmp.Diff(want, station); diff != "" {
		t.Errorf("TSRT mismatch (-want +got):\n%s", diff)
	}

	wantFirst := readingOutput{
		Pollutant: "PM10", Value: "51.8833333333333", Unit: "µg/m³",
		Time: "2026-11-05T05:00:00+04:00",
	}
	if first != wantFirst || len(out.Stations[index].Readings) != 6 {
		t.Errorf("first reading = %+v of %d, want %+v of 6",
			first, len(out.Stations[index].Readings), wantFirst)
	}

	georgian := quality(t, session, map[string]any{keyCity: tbilisi, keyLanguage: "ka"})
	if georgian.Stations[0].City != "თბილისი" {
		t.Errorf("city in ka = %q, want თბილისი", georgian.Stations[0].City)
	}
}

// NEAAIR-SC-019: A failing NEA with an empty cache gives the message of the outage.
func TestAFailingNEA(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeNEA(t, http.StatusInternalServerError).config(t),
		slog.New(slog.DiscardHandler))

	want := "The NEA did not answer. Try again in a few minutes."

	for _, name := range []string{getQuality, listHistory} {
		result := call(t, session, name, map[string]any{keyStation: tsrt})

		text, ok := result.Content[0].(*mcp.TextContent)
		if !result.IsError || !ok || text.Text != want {
			t.Errorf("%s = %+v, want the tool error %q", name, result.Content, want)
		}
	}
}

// NEAAIR-SC-025: Three cities and a station in two languages send one call of each method.
func TestQuestionsShareOneRead(t *testing.T) {
	t.Parallel()

	fake := newFakeNEA(t, http.StatusOK)
	session := serve(t, fake.config(t), slog.New(slog.DiscardHandler))

	for _, language := range []string{"en", "ka"} {
		quality(t, session, map[string]any{keyCity: tbilisi, keyLanguage: language})
		quality(t, session, map[string]any{keyCity: "Batumi", keyLanguage: language})
		quality(t, session, map[string]any{keyStation: tsrt, keyLanguage: language})
	}

	if hourly, cities := fake.count(
		"/api/get_data_1hour/",
	), fake.count(
		"/api/municipalities/",
	); hourly != 1 ||
		cities != 1 {
		t.Errorf("calls = %d hourly, %d municipalities, want 1 and 1", hourly, cities)
	}
}
