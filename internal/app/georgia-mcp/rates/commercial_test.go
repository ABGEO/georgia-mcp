package rates_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
)

const (
	tbcTestdata   = "../../../../pkg/tbc/testdata"
	getCommercial = "rates_get_commercial_rates"
	tbcKey        = "GEOMCP_RATES_TBC_API_KEY"
)

type commercialRateOutput struct {
	Currency string `json:"currency"`
	Buy      string `json:"buy"`
	Sell     string `json:"sell"`
}

type bankRatesOutput struct {
	Bank   string                 `json:"bank"`
	Source string                 `json:"source"`
	AsOf   string                 `json:"as_of"`
	Stale  bool                   `json:"stale"`
	Rates  []commercialRateOutput `json:"rates"`
}

type commercialOutput struct {
	Meaning string            `json:"meaning"`
	Banks   []bankRatesOutput `json:"banks"`
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

func debugLogger(sink *syncBuffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(sink, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func tbcRecorded(t *testing.T, status int, name string) http.HandlerFunc {
	t.Helper()

	path := filepath.Join(tbcTestdata, name)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

// commercialConfig points TBC at the vendor. The NBG is never called.
func commercialConfig(t *testing.T, vendor http.Handler, key string) rates.Config {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	tbcURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	nbgURL, _ := url.Parse("https://nbg.invalid")

	return rates.Config{
		NBG: nbgConfig(*nbgURL),
		TBC: rates.TBCConfig{
			APIKey:  key,
			BaseURL: *tbcURL,
			HTTP:    rates.TBCHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
		},
	}
}

// TBCRATE-SC-018: The answer states the meaning of buy and sell, and gives the bank, its
// read instant, and the rates.
func TestGetCommercialRatesOfTBC(t *testing.T) {
	t.Parallel()

	cfg := commercialConfig(t, tbcRecorded(t, http.StatusOK, "commercial.json"), "test-key")
	session := serve(t, cfg, slog.New(slog.DiscardHandler))

	out := decode[commercialOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar},
	}))

	want := commercialOutput{
		Meaning: "buy: the price in GEL that the bank pays when you sell it one unit. " +
			"sell: the price in GEL that the bank asks when you buy one unit from it.",
		Banks: []bankRatesOutput{{
			Bank:   "tbc",
			Source: "TBC Bank",
			AsOf:   "2026-10-18T14:00:00+04:00",
			Rates:  []commercialRateOutput{{Currency: dollar, Buy: "2.566", Sell: "2.628"}},
		}},
	}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// TBCRATE-SC-019: The key appears in no tool error and in no log record.
func TestGetCommercialRatesKeepsTheKeySecret(t *testing.T) {
	t.Parallel()

	const key = "secret-tbc-key"

	var sentKey string

	problem := tbcRecorded(t, http.StatusUnauthorized, "problem-401.json")
	vendor := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentKey = r.Header.Get("Apikey")
		problem(w, r)
	})

	sink := &syncBuffer{}
	session := serve(t, commercialConfig(t, vendor, key), debugLogger(sink))

	result := call(t, session, getCommercial, map[string]any{argCurrencies: []string{dollar}})

	if !result.IsError || strings.Contains(text(result), key) {
		t.Errorf("result = %q, want a tool error with no key", text(result))
	}

	if logs := sink.String(); strings.Contains(logs, key) {
		t.Errorf("the log holds the key:\n%s", logs)
	}

	if sentKey != key {
		t.Errorf("the vendor got the key %q, want %q", sentKey, key)
	}
}

// TBCRATE-SC-020: With a key, the server offers the commercial tool. With none, it offers
// the official tools only, and warns once.
func TestTheCommercialToolNeedsAKey(t *testing.T) {
	t.Parallel()

	official := []string{getRates, "rates_list_official_history"}

	tests := []struct {
		name         string
		key          string
		wantTools    []string
		wantWarnings int
	}{
		{name: "key", key: "test-key", wantTools: append(slices.Clone(official), getCommercial)},
		{name: "no key", key: "", wantTools: official, wantWarnings: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sink := &syncBuffer{}
			session := serve(
				t,
				commercialConfig(t, http.NotFoundHandler(), tt.key),
				debugLogger(sink),
			)

			names := toolNames(t, session)
			if diff := cmp.Diff(slices.Sorted(slices.Values(tt.wantTools)), names); diff != "" {
				t.Errorf("tools mismatch (-want +got):\n%s", diff)
			}

			warnings := warningsOf(sink)
			if len(warnings) != tt.wantWarnings {
				t.Fatalf("warnings = %q, want %d", warnings, tt.wantWarnings)
			}

			for _, warning := range warnings {
				if !strings.Contains(warning, tbcKey) {
					t.Errorf("warning = %q, want the variable %s", warning, tbcKey)
				}
			}
		})
	}
}

func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()

	list, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	names := make([]string, 0, len(list.Tools))
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
	}

	return slices.Sorted(slices.Values(names))
}

func warningsOf(sink *syncBuffer) []string {
	var warnings []string

	for line := range strings.Lines(sink.String()) {
		if strings.Contains(line, `"level":"WARN"`) {
			warnings = append(warnings, line)
		}
	}

	return warnings
}
