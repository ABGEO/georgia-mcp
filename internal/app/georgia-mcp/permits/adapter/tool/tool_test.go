package tool_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/cache"
	tasadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/tas"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	tasclient "github.com/abgeo/georgia-mcp/pkg/tas"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata    = "../../../../../../pkg/tas/testdata/"
	search      = "permits_search_applications"
	get         = "permits_get_application"
	parcelCode  = "01.18.04.020.005"
	approvedNo  = "AR11168188"
	keyCode     = "code"
	keyFrom     = "from"
	keyTo       = "to"
	keyLimit    = "limit"
	keyNumber   = "number"
	keyLanguage = "language"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// vendor is the fake TAS: it serves the recorded answers, counts the calls, and can fail
// or wait.
type vendor struct {
	status int
	delay  time.Duration
	calls  atomic.Int32
}

func (v *vendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.calls.Add(1)
	time.Sleep(v.delay)

	if v.status != 0 {
		w.WriteHeader(v.status)

		return
	}

	sent, _ := io.ReadAll(r.Body)

	body, err := os.ReadFile(testdata + file(string(sent)))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	_, _ = w.Write(body)
}

// file picks the answer: the approved number, the parcel, a month of 2128 applications
// for any period, and nothing for the rest.
func file(body string) string {
	switch {
	case strings.Contains(body, "string:"+approvedNo+"\n"):
		return "search-number.js"
	case strings.Contains(body, "string:"+parcelCode+"\n"):
		return "search-code.js"
	case strings.Contains(body, "fromDate:"):
		return "search-month.js"
	default:
		return "search-empty.js"
	}
}

// connect serves the fake TAS through the real client, adapter, and cache.
func connect(t *testing.T, fake *vendor) *mcp.ClientSession {
	t.Helper()

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	adapter := tasadapter.NewAdapter(tasclient.New(vendorhttp.WithBaseURL(server.URL)), clock)

	sources, err := cache.NewSourceCache(adapter)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = sources.Close() })

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		SearchApplications: app.NewSearchApplications(sources),
		GetApplication:     app.NewGetApplication(sources),
	}); err != nil {
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

func decode[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	if result.IsError {
		t.Fatalf("CallTool() = %+v, want a result", result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var output T
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatal(err)
	}

	return output
}

func message(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool() = %+v, want an error", result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Content[0] = %T, want text", result.Content[0])
	}

	return text.Text
}

type code struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type application struct {
	Number       string `json:"number"`
	URL          string `json:"url"`
	Registered   string `json:"registered"`
	Status       code   `json:"status"`
	DecisionText string `json:"decision_text"`
}

type searchResult struct {
	Source       string        `json:"source"`
	AsOf         string        `json:"as_of"`
	Total        int           `json:"total"`
	Applications []application `json:"applications"`
}

type getResult struct {
	application

	Source string `json:"source"`
}

// TASPERM-SC-014: A search by code gives its source, its total, and the applications
// newest first, with no decision text.
func TestSearchApplications(t *testing.T) {
	t.Parallel()

	got := decode[searchResult](
		t,
		call(t, connect(t, &vendor{}), search, map[string]any{keyCode: parcelCode}),
	)

	if got.Source != "TAS" || !strings.HasSuffix(got.AsOf, "+04:00") || got.Total != 5 ||
		len(got.Applications) != 5 {
		t.Fatalf(
			"search = %s, %s, %d of %d",
			got.Source,
			got.AsOf,
			len(got.Applications),
			got.Total,
		)
	}

	dates := make([]string, 0, len(got.Applications))

	for _, item := range got.Applications {
		dates = append(dates, item.Registered)

		if item.DecisionText != "" || !strings.HasPrefix(item.URL, "https://tas.ge/") {
			t.Errorf("application %s = %+v, want a page and no decision text", item.Number, item)
		}
	}

	if !slices.IsSortedFunc(dates, func(a, b string) int { return strings.Compare(b, a) }) {
		t.Errorf("registered = %v, want newest first", dates)
	}
}

// TASPERM-SC-014: In ka, a status takes the name of the page of TAS.
func TestSearchApplicationsInGeorgian(t *testing.T) {
	t.Parallel()

	got := decode[searchResult](t, call(t, connect(t, &vendor{}), search,
		map[string]any{keyCode: parcelCode, keyLanguage: "ka"}))

	names := map[string]string{}
	for _, item := range got.Applications {
		names[item.Status.Code] = item.Status.Name
	}

	if names["approved"] != "თანხმობა" {
		t.Errorf("names = %v, want approved named თანხმობა", names)
	}
}

// TASPERM-SC-015: A number with no prefix gives its application with its decision.
func TestGetApplication(t *testing.T) {
	t.Parallel()

	got := decode[getResult](
		t,
		call(t, connect(t, &vendor{}), get, map[string]any{keyNumber: "1168188"}),
	)

	want := code{Code: "approved", Name: "Approved"}
	if got.Source != "TAS" || got.Number != approvedNo || got.Status != want ||
		got.URL != "https://tas.ge/?p=publicpage&documentId=1168188" || got.DecisionText == "" {
		t.Errorf("get = %+v", got)
	}
}

// TASPERM-SC-016: Each wrong input gives its message.
func TestInputErrors(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{})

	for _, tt := range inputCases() {
		if got := message(t, call(t, session, tt.tool, tt.args)); got != tt.want {
			t.Errorf("%s %v = %q, want %q", tt.tool, tt.args, got, tt.want)
		}
	}
}

type inputCase struct {
	tool string
	args map[string]any
	want string
}

func inputCases() []inputCase {
	return []inputCase{
		{
			search,
			map[string]any{keyCode: parcelCode, keyLanguage: "ru"},
			`language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{
			search,
			map[string]any{keyCode: parcelCode, keyLimit: 60},
			"limit: 60 is out of range. Use 1 to 50.",
		},
		{
			search,
			map[string]any{keyCode: "01.18"},
			`code: "01.18" is not a cadastral code. Use 4 or 5 groups of digits with dots, as 01.18.04.020.005.`,
		},
		{search, map[string]any{keyFrom: "4 Nov"}, `from: "4 Nov" is not a date. Use YYYY-MM-DD.`},
		{
			search,
			map[string]any{keyFrom: "2026-11-10", keyTo: "2026-11-04"},
			"from, to: 2026-11-10 is after 2026-11-04. Give the earlier date as from.",
		},
		{
			search,
			map[string]any{keyFrom: "2026-01-01", keyTo: "2026-03-01"},
			"from, to: 2026-01-01 to 2026-03-01 is 60 days. Use at most 31 days.",
		},
		{
			search,
			map[string]any{},
			"code, from, to: empty. Give a cadastral code, a period, or both.",
		},
		{
			search,
			map[string]any{keyFrom: "2026-10-01", keyTo: "2026-10-31"},
			"TAS has 2128 applications for this search. The server sorts at most 500. " +
				"Give a shorter period, or a cadastral code.",
		},
		{
			get,
			map[string]any{keyNumber: "AR1116818X"},
			`number: "AR1116818X" is not an application number. Use AR1 and digits, as AR11168188.`,
		},
		{
			get,
			map[string]any{keyNumber: "AR19999999"},
			"number: TAS has no application AR19999999. Find the number with permits_search_applications.",
		},
	}
}

// TASPERM-SC-017: A failing TAS with an empty cache gives the message of the outage.
func TestAFailingTAS(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{status: http.StatusInternalServerError})
	want := "TAS did not answer. Try again in a few minutes."

	for name, args := range map[string]map[string]any{
		search: {keyCode: parcelCode},
		get:    {keyNumber: approvedNo},
	} {
		if got := message(t, call(t, session, name, args)); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

// TASPERM-SC-020: Each tool sends one call, and the same calls again send none.
func TestCallsToTAS(t *testing.T) {
	t.Parallel()

	fake := &vendor{}
	session := connect(t, fake)

	steps := []struct {
		tool string
		args map[string]any
	}{
		{search, map[string]any{keyCode: parcelCode}},
		{get, map[string]any{keyNumber: approvedNo}},
	}

	for _, again := range []int32{1, 0} {
		for _, step := range steps {
			before := fake.calls.Load()

			decode[map[string]any](t, call(t, session, step.tool, step.args))

			if got := fake.calls.Load() - before; got != again {
				t.Errorf("%s sent %d calls, want %d", step.tool, got, again)
			}
		}
	}
}

// TASPERM-SC-023: With 4 seconds for each call, each tool answers in less than 10 seconds.
func TestAnswerTimes(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{delay: 4 * time.Second})

	for name, args := range map[string]map[string]any{
		search: {keyCode: parcelCode},
		get:    {keyNumber: approvedNo},
	} {
		start := time.Now()

		decode[map[string]any](t, call(t, session, name, args))

		if elapsed := time.Since(start); elapsed >= 10*time.Second {
			t.Errorf("%s took %v, want less than 10s", name, elapsed)
		}
	}
}
