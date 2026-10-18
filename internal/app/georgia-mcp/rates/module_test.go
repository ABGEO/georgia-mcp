package rates_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	testdata = "../../../../pkg/nbg/testdata"
	getRates = "rates_get_official_rates"
	// The recorded answers date from 2026-10-17. The clock stands one day later.
	now = "2026-10-18T10:00:00Z"

	recordedDate = "2026-10-17"
	saturday     = "2026-10-10"
	euro         = "EUR"
	dollar       = "USD"

	argCurrencies = "currencies"
	argDate       = "date"
)

type rateOutput struct {
	Currency    string `json:"currency"`
	Name        string `json:"name"`
	Rate        string `json:"rate"`
	RatePerUnit string `json:"rate_per_unit"`
	ValidFrom   string `json:"valid_from"`
}

type ratesOutput struct {
	ValidFrom string       `json:"valid_from"`
	Rates     []rateOutput `json:"rates"`
	Missing   []string     `json:"missing"`
}

type historyOutput struct {
	Name   string       `json:"name"`
	Points []rateOutput `json:"points"`
}

// recordedVendor serves the recorded answer of the language and the date of a request.
func recordedVendor(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		language, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/currencies/"), "/")
		path := filepath.Join(testdata, language+"-"+r.URL.Query().Get("date")+".json")

		body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
		if err != nil {
			http.Error(w, "no recorded answer", http.StatusNotFound)

			return
		}

		_, _ = w.Write(body) //nolint:gosec // The test server writes a recorded answer.
	}
}

func connect(t *testing.T, vendor http.Handler) *mcp.ClientSession {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	cfg := rates.Config{NBG: rates.NBGConfig{
		BaseURL:      *baseURL,
		CacheEntries: 512,
		HTTP:         httpclient.Config{Timeout: 10 * time.Second, Rate: 5, Burst: 5},
	}}

	instant, _ := time.Parse(time.RFC3339, now)

	module, err := rates.NewWithClock(
		cfg,
		slog.New(slog.DiscardHandler),
		func() time.Time { return instant },
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = module.Close() })

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
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
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}

	return result
}

func decode[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	if result.IsError {
		t.Fatalf("result is an error: %v", text(result))
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

func text(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}

	if content, ok := result.Content[0].(*mcp.TextContent); ok {
		return content.Text
	}

	return ""
}

func ratesOf(out ratesOutput) map[string]string {
	got := map[string]string{}
	for _, rate := range out.Rates {
		got[rate.Currency] = rate.Rate
	}

	return got
}

// NBGRATE-SC-001: The rates of USD and EUR on 2026-10-17.
func TestGetRatesOfTwoCurrencies(t *testing.T) {
	t.Parallel()

	out := decode[ratesOutput](t, call(t, connect(t, recordedVendor(t)), getRates, map[string]any{
		argCurrencies: []string{dollar, euro}, argDate: recordedDate,
	}))

	if diff := cmp.Diff(
		map[string]string{dollar: "2.5982", euro: "2.9274"},
		ratesOf(out),
	); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}

	if len(out.Missing) != 0 {
		t.Errorf("missing = %v, want none", out.Missing)
	}
}

// NBGRATE-SC-004: No currency gives the 42 rates of the set, sorted.
func TestGetRatesOfEveryCurrency(t *testing.T) {
	t.Parallel()

	out := decode[ratesOutput](
		t,
		call(t, connect(t, recordedVendor(t)), getRates, map[string]any{argDate: recordedDate}),
	)

	if len(out.Rates) != 42 || out.Rates[0].Currency != "AED" || out.Rates[41].Currency != "ZAR" {
		t.Fatalf(
			"rates = %d, from %v, want 42 from AED to ZAR",
			len(out.Rates),
			out.Rates[0].Currency,
		)
	}
}

// NBGRATE-SC-006: A Sunday gives the set valid from the Saturday before.
func TestGetRatesOfASunday(t *testing.T) {
	t.Parallel()

	out := decode[ratesOutput](t, call(t, connect(t, recordedVendor(t)), getRates, map[string]any{
		argCurrencies: []string{dollar}, argDate: "2026-10-11",
	}))

	if out.ValidFrom != saturday || len(out.Rates) != 1 ||
		out.Rates[0].ValidFrom != saturday {
		t.Errorf("valid_from = %q, rates = %v, want 2026-10-10", out.ValidFrom, out.Rates)
	}
}

// NBGRATE-SC-011: A date before the first set gives every code as missing.
func TestGetRatesBeforeTheFirstSet(t *testing.T) {
	t.Parallel()

	out := decode[ratesOutput](t, call(t, connect(t, recordedVendor(t)), getRates, map[string]any{
		argCurrencies: []string{dollar, euro}, argDate: "1990-01-01",
	}))

	if len(out.Rates) != 0 || out.ValidFrom != "" {
		t.Errorf("rates = %v, valid_from = %q, want none", out.Rates, out.ValidFrom)
	}

	if diff := cmp.Diff([]string{euro, dollar}, out.Missing); diff != "" {
		t.Errorf("missing mismatch (-want +got):\n%s", diff)
	}
}

// NBGRATE-SC-014: The language picks the names. An unknown language is a tool error.
func TestGetRatesInALanguage(t *testing.T) {
	t.Parallel()

	session := connect(t, recordedVendor(t))

	tests := []struct {
		language any
		want     string
	}{
		{language: "ka", want: "აშშ დოლარი"},
		{language: nil, want: "US Dollar"},
	}

	for _, tt := range tests {
		args := map[string]any{argCurrencies: []string{dollar}, argDate: recordedDate}
		if tt.language != nil {
			args["language"] = tt.language
		}

		out := decode[ratesOutput](t, call(t, session, getRates, args))
		if len(out.Rates) != 1 || out.Rates[0].Name != tt.want {
			t.Errorf("language %v: rates = %v, want the name %q", tt.language, out.Rates, tt.want)
		}
	}

	result := call(t, session, getRates, map[string]any{"language": "de"})
	if !result.IsError ||
		!strings.HasPrefix(text(result), `language: "de" is not a language of the NBG.`) {
		t.Errorf("language de: result = %q, want a tool error", text(result))
	}
}

// NBGRATE-SC-020: An outage with nothing in the cache tells the model to try again.
func TestGetRatesWhenTheNBGIsDown(t *testing.T) {
	t.Parallel()

	down := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	})

	result := call(t, connect(t, down), getRates, map[string]any{argDate: recordedDate})

	if want := "The NBG did not answer. Try again in a few minutes."; !result.IsError ||
		text(result) != want {
		t.Errorf("result = %q, want the tool error %q", text(result), want)
	}
}

// NBGRATE-SC-032: The history of a week holds each set in force, from the Saturday before.
func TestListHistoryOfAWeek(t *testing.T) {
	t.Parallel()

	out := decode[historyOutput](
		t,
		call(t, connect(t, recordedVendor(t)), "rates_list_official_history", map[string]any{
			"currency": dollar, "from": "2026-10-11", "to": recordedDate,
		}),
	)

	validFrom := make([]string, 0, len(out.Points))
	for _, point := range out.Points {
		validFrom = append(validFrom, point.ValidFrom)
	}

	want := []string{saturday, "2026-10-13", "2026-10-14", "2026-10-16", recordedDate}
	if diff := cmp.Diff(want, validFrom); diff != "" {
		t.Errorf("points mismatch (-want +got):\n%s", diff)
	}

	if out.Name != "US Dollar" {
		t.Errorf("name = %q, want US Dollar", out.Name)
	}
}

// NBGRATE-SC-026: A history of 31 days with an empty cache answers within 10 seconds.
func TestListHistoryOfAMonthIsFast(t *testing.T) {
	t.Parallel()

	template, err := os.ReadFile(filepath.Join(testdata, "en-2026-10-17.json"))
	if err != nil {
		t.Fatal(err)
	}

	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)

		date, _ := time.Parse(time.DateOnly, r.URL.Query().Get("date"))
		if date.Weekday() == time.Sunday {
			date = date.AddDate(0, 0, -1)
		}

		validFrom := date.Format(time.DateOnly) + "T00:00:00"
		body := strings.ReplaceAll(string(template), "2026-10-17T00:00:00", validFrom)

		_, _ = w.Write([]byte(body)) //nolint:gosec // The test server writes a recorded answer.
	})

	session := connect(t, slow)
	start := time.Now()

	out := decode[historyOutput](t, call(t, session, "rates_list_official_history", map[string]any{
		"currency": dollar, "from": "2026-09-17", "to": recordedDate,
	}))

	if elapsed := time.Since(start); elapsed >= 10*time.Second {
		t.Errorf("the history took %v, want less than 10s", elapsed)
	}

	if len(out.Points) != 27 {
		t.Errorf("points = %d, want 27: 31 days less 4 Sundays", len(out.Points))
	}
}
