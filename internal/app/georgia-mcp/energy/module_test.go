package energy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	testdata  = "../../../../pkg/gse/testdata/"
	getGrid   = "energy_get_grid"
	keyMoment = "moment"
	grid1200  = "GSE_EXPORT_NEW_PERIOD_03M_16112026_1200.XML"
	freq1200  = "GSE_EXPORT_PERIOD_01M_16112026_1200.XML"

	noonSlot     = "2026-11-16T12:00:00+04:00"
	imported     = "import"
	measuredCode = "ACT"
	zero         = "0.000000"
	noDirection  = "none"
)

func at(minute, second int) func() time.Time {
	return func() time.Time {
		return time.Date(2026, time.November, 16, 12, minute, second, 0, kernel.TbilisiZone())
	}
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

// fakeGSE serves the recorded files by name, the 404 page of nginx for any other name,
// or the status for every name, and counts the calls of each name.
type fakeGSE struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  map[string]int
}

func newFakeGSE(t *testing.T, status int, names ...string) *fakeGSE {
	t.Helper()

	read := func(name string) []byte {
		body, err := os.ReadFile(testdata + name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		return body
	}

	recorded := map[string][]byte{
		grid1200: read("grid-16112026-1200.xml"),
		freq1200: read("frequency-16112026-1200.xml"),
	}
	notFound := read("not-found.html")

	files := map[string][]byte{}
	for _, name := range names {
		files[name] = recorded[name]
	}

	fake := &fakeGSE{calls: map[string]int{}}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Base(r.URL.Path)

		fake.mu.Lock()
		fake.calls[name]++
		fake.mu.Unlock()

		body, ok := files[name]

		switch {
		case status != http.StatusOK:
			w.WriteHeader(status)
		case !ok:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(notFound)
		default:
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakeGSE) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[name]
}

func (f *fakeGSE) config(t *testing.T) energy.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return energy.Config{GSE: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 20},
	}}
}

func serve(
	t *testing.T,
	cfg energy.Config,
	logger *slog.Logger,
	now func() time.Time,
) *mcp.ClientSession {
	t.Helper()

	module, err := energy.NewWithClock(cfg, logger, now)
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

func discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func call(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(
		t.Context(),
		&mcp.CallToolParams{Name: getGrid, Arguments: args},
	)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	return result
}

func errorText(t *testing.T, session *mcp.ClientSession, args map[string]any) string {
	t.Helper()

	result := call(t, session, args)
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool(%v) = %+v, want a tool error", args, result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool() content = %T, want text", result.Content[0])
	}

	return text.Text
}

type valueOutput struct {
	Value   string `json:"value"`
	Quality string `json:"quality"`
	Meaning string `json:"meaning"`
}

type lineOutput struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Direction string `json:"direction"`
	Quality   string `json:"quality"`
	Meaning   string `json:"meaning"`
}

type flowOutput struct {
	Country   string       `json:"country"`
	Value     string       `json:"value"`
	Direction string       `json:"direction"`
	Quality   string       `json:"quality"`
	Meaning   string       `json:"meaning"`
	Lines     []lineOutput `json:"lines"`
}

type gridOutput struct {
	Source         string `json:"source"`
	AsOf           string `json:"as_of"`
	Stale          bool   `json:"stale"`
	Time           string `json:"time"`
	AverageMinutes int    `json:"average_minutes"`
	Generation     struct {
		Hydro valueOutput `json:"hydro"`
	} `json:"generation"`
	Consumption valueOutput  `json:"consumption"`
	Flows       []flowOutput `json:"flows"`
	NetImport   struct {
		Value     string `json:"value"`
		Direction string `json:"direction"`
	} `json:"net_import"`
	Measured  int `json:"measured"`
	Values    int `json:"values"`
	Frequency *struct {
		Value   string `json:"value"`
		Time    string `json:"time"`
		Quality string `json:"quality"`
		Meaning string `json:"meaning"`
	} `json:"frequency"`
}

func gridOf(t *testing.T, session *mcp.ClientSession, args map[string]any) gridOutput {
	t.Helper()

	result := call(t, session, args)
	if result.IsError {
		t.Fatalf("CallTool() = %+v, want a result", result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out gridOutput
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

// GSEGRID-SC-018: With GSE on, the server offers the read-only tool energy_get_grid.
func TestTheSwitchDecidesTheTools(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeGSE(t, http.StatusOK).config(t), discard(), at(1, 30))

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(tools.Tools) != 1 || tools.Tools[0].Name != getGrid ||
		!readOnly(tools.Tools[0].Annotations) {
		t.Errorf("tools = %+v, want the read-only tool %s", tools.Tools, getGrid)
	}
}

// GSEGRID-SC-018: With GSE off, the server offers no tool of energy and says so.
func TestTheSwitchTurnsGSEOff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	off := serve(t, energy.Config{}, slog.New(slog.NewJSONHandler(sink, nil)), at(1, 30))

	none, err := off.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(none.Tools) != 0 || !strings.Contains(sink.String(), `"msg":"GSE is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_ENERGY_GSE_ENABLED"`) {
		t.Errorf("tools = %d, log = %s, want no tool and the record GSE is off",
			len(none.Tools), sink)
	}
}

// GSEGRID-SC-019: The newest grid state reads the recorded files through the whole
// context.
func TestGetGridReadsGSE(t *testing.T) {
	t.Parallel()

	fake := newFakeGSE(t, http.StatusOK, grid1200, freq1200)
	out := gridOf(t, serve(t, fake.config(t), discard(), at(1, 30)), nil)

	head := []any{
		out.Source,
		out.AsOf,
		out.Stale,
		out.Time,
		out.AverageMinutes,
		out.Measured,
		out.Values,
	}
	wantHead := []any{
		"Georgian State Electrosystem (GSE)", "2026-11-16T12:01:30+04:00", false,
		noonSlot, 3, 5, 13,
	}

	if diff := cmp.Diff(wantHead, head); diff != "" {
		t.Errorf("head mismatch (-want +got):\n%s", diff)
	}

	wantHydro := valueOutput{
		Value:   "1331.362305",
		Quality: "SUB",
		Meaning: "substituted by the operator",
	}
	if out.Generation.Hydro != wantHydro || out.Consumption.Value != "1843.806030" {
		t.Errorf("hydro = %+v, consumption = %+v", out.Generation.Hydro, out.Consumption)
	}

	if diff := cmp.Diff(wantFlows(), out.Flows); diff != "" {
		t.Errorf("flows mismatch (-want +got):\n%s", diff)
	}

	if out.NetImport.Value != "332.358163" || out.NetImport.Direction != imported {
		t.Errorf("net_import = %+v, want 332.358163 import", out.NetImport)
	}

	checkFrequency(t, out)
}

func checkFrequency(t *testing.T, out gridOutput) {
	t.Helper()

	if out.Frequency == nil || out.Frequency.Value != "49.990002" ||
		out.Frequency.Time != noonSlot || out.Frequency.Quality != measuredCode {
		t.Errorf("frequency = %+v, want 49.990002 ACT at 12:00", out.Frequency)
	}
}

func wantFlows() []flowOutput {
	measured := "measured"

	return []flowOutput{
		{
			Country: "Azerbaijan", Value: "20.631090", Direction: imported,
			Quality: measuredCode, Meaning: measured,
		},
		{
			Country: "Armenia", Value: zero, Direction: noDirection,
			Quality: "SUB", Meaning: "substituted by the operator",
		},
		{Country: "Russia", Value: "32.491386", Direction: imported, Lines: []lineOutput{
			{
				Name:      "Russia",
				Value:     zero,
				Direction: noDirection,
				Quality:   measuredCode,
				Meaning:   measured,
			},
			{
				Name:      "Salkhino",
				Value:     zero,
				Direction: noDirection,
				Quality:   measuredCode,
				Meaning:   measured,
			},
			{
				Name: "Java", Value: "18.299999", Direction: imported,
				Quality: "BLS", Meaning: "not documented by GSE",
			},
			{
				Name:      "Nakaduli",
				Value:     "14.191387",
				Direction: imported,
				Quality:   measuredCode,
				Meaning:   measured,
			},
		}},
		{
			Country: "Turkey", Value: "279.235687", Direction: imported,
			Quality: measuredCode, Meaning: measured,
		},
	}
}

// GSEGRID-SC-020: A moment whose slot GSE has not published gives the slot before.
func TestGetGridSearchesBack(t *testing.T) {
	t.Parallel()

	fake := newFakeGSE(t, http.StatusOK, grid1200, freq1200)
	out := gridOf(t, serve(t, fake.config(t), discard(), at(5, 0)),
		map[string]any{keyMoment: "2026-11-16 12:04"})

	if out.Time != noonSlot {
		t.Errorf("time = %q, want 2026-11-16T12:00:00+04:00", out.Time)
	}
}

// GSEGRID-SC-021: A moment in UTC gives the slot of Tbilisi. A missing frequency leaves
// the grid state.
func TestGetGridReadsAMomentInUTC(t *testing.T) {
	t.Parallel()

	fake := newFakeGSE(t, http.StatusOK, grid1200)
	out := gridOf(t, serve(t, fake.config(t), discard(), at(1, 30)),
		map[string]any{keyMoment: "2026-11-16T08:01:00Z"})

	if out.Time != noonSlot || out.Frequency != nil {
		t.Errorf("time = %q, frequency = %+v, want 12:00 and no frequency", out.Time, out.Frequency)
	}
}

// GSEGRID-SC-022: Each bad moment of 4.5 gives its message.
func TestGetGridRefusesABadMoment(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeGSE(t, http.StatusOK).config(t), discard(), at(1, 30))

	tests := []struct {
		moment string
		want   string
	}{
		{
			moment: "yesterday",
			want: `moment: "yesterday" is not a moment. Use 2026-11-16T13:00 for a time of ` +
				`Tbilisi, or RFC 3339 with an offset.`,
		},
		{
			moment: "2026-11-16 13:01:30",
			want: "moment: 2026-11-16T13:01:30+04:00 is in the future. Give a moment from " +
				"2026-10-16T12:01:30+04:00 to now.",
		},
		{
			moment: "2026-10-07 12:00",
			want: "moment: 2026-10-07T12:00:00+04:00 is more than 31 days ago. Give a moment " +
				"from 2026-10-16T12:01:30+04:00 to now.",
		},
		{
			moment: "2026-11-16 10:05",
			want: "GSE published no grid state in the 30 minutes up to " +
				"2026-11-16T10:03:00+04:00. Try another moment.",
		},
	}

	for _, tt := range tests {
		if got := errorText(t, session, map[string]any{keyMoment: tt.moment}); got != tt.want {
			t.Errorf("moment %q = %q, want %q", tt.moment, got, tt.want)
		}
	}
}

// GSEGRID-SC-023: A failing GSE with an empty cache gives the message of the outage.
func TestAFailingGSE(t *testing.T) {
	t.Parallel()

	session := serve(
		t,
		newFakeGSE(t, http.StatusInternalServerError).config(t),
		discard(),
		at(1, 30),
	)

	want := "GSE did not answer. Try again in a few minutes."
	if got := errorText(t, session, nil); got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// GSEGRID-SC-024: Two calls of the newest grid state send one grid call and one frequency
// call to GSE.
func TestCallsShareOneRead(t *testing.T) {
	t.Parallel()

	fake := newFakeGSE(t, http.StatusOK, grid1200, freq1200)
	session := serve(t, fake.config(t), discard(), at(1, 30))

	gridOf(t, session, nil)
	gridOf(t, session, map[string]any{})

	if grid, frequency := fake.count(grid1200), fake.count(freq1200); grid != 1 || frequency != 1 {
		t.Errorf("calls = %d grid, %d frequency, want 1 and 1", grid, frequency)
	}
}
