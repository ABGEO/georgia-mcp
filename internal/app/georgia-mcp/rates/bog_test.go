package rates_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
)

const (
	bogTestdata = "../../../../pkg/bog/testdata"
	bogSwitch   = "GEOMCP_RATES_BOG_ENABLED"
	argBanks    = "banks"
	bogCode     = "bog"
)

type bogBankOutput struct {
	Bank    string                 `json:"bank"`
	Source  string                 `json:"source"`
	AsOf    string                 `json:"as_of"`
	Rates   []commercialRateOutput `json:"rates"`
	Missing []string               `json:"missing"`
}

type bogOutput struct {
	Banks []bogBankOutput `json:"banks"`
}

// bogVendor serves commercial-usd.json for USD, and 204 for any other code.
func bogVendor(t *testing.T) http.HandlerFunc {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(bogTestdata, "commercial-usd.json"))
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if path.Base(r.URL.Path) != dollar {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		_, _ = w.Write(body)
	}
}

// withBOG turns Bank of Georgia on and points it at the vendor.
func withBOG(t *testing.T, cfg rates.Config, vendor http.Handler) rates.Config {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	bogURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	cfg.BOG = rates.BOGConfig{
		Enabled: true,
		BaseURL: *bogURL,
		HTTP:    rates.BOGHTTPConfig{Timeout: 10 * time.Second, Rate: 0.8, Burst: 10},
	}

	return cfg
}

// BOGRATE-SC-011: The answer gives Bank of Georgia, its read instant, and the rate of
// USD with no padding zero.
func TestGetCommercialRatesOfBankOfGeorgia(t *testing.T) {
	t.Parallel()

	cfg := withBOG(t, commercialConfig(t, http.NotFoundHandler(), ""), bogVendor(t))
	session := serve(t, cfg, debugLogger(&syncBuffer{}))

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar},
	}))

	want := bogOutput{Banks: []bogBankOutput{{
		Bank:   bogCode,
		Source: "Bank of Georgia",
		AsOf:   "2026-10-18T14:00:00+04:00",
		Rates:  []commercialRateOutput{{Currency: dollar, Buy: "2.565", Sell: "2.629"}},
	}}}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// BOGRATE-SC-012: With no bank named, the answer holds Bank of Georgia, then TBC.
func TestGetCommercialRatesOfEveryBank(t *testing.T) {
	t.Parallel()

	tbc := commercialConfig(t, tbcRecorded(t, http.StatusOK, "commercial.json"), "test-key")
	session := serve(t, withBOG(t, tbc, bogVendor(t)), debugLogger(&syncBuffer{}))

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar},
	}))

	banks := make([]string, 0, len(out.Banks))
	for _, bank := range out.Banks {
		banks = append(banks, bank.Bank)
	}

	if diff := cmp.Diff([]string{bogCode, "tbc"}, banks); diff != "" {
		t.Errorf("banks mismatch (-want +got):\n%s", diff)
	}
}

// BOGRATE-SC-013: A code that Bank of Georgia answers with 204 is missing.
func TestGetCommercialRatesOfBankOfGeorgiaNamesAMissingCode(t *testing.T) {
	t.Parallel()

	cfg := withBOG(t, commercialConfig(t, http.NotFoundHandler(), ""), bogVendor(t))
	session := serve(t, cfg, debugLogger(&syncBuffer{}))

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar, "GEL"},
	}))

	if len(out.Banks) != 1 {
		t.Fatalf("banks = %+v, want bog only", out.Banks)
	}

	bank := out.Banks[0]
	if len(bank.Rates) != 1 || bank.Rates[0].Currency != dollar {
		t.Errorf("rates = %+v, want USD only", bank.Rates)
	}

	if diff := cmp.Diff([]string{"GEL"}, bank.Missing); diff != "" {
		t.Errorf("missing mismatch (-want +got):\n%s", diff)
	}
}

// BOGRATE-SC-014: The switch of Bank of Georgia decides whether the server offers it.
// Off, the server logs one info record that names the variable.
func TestTheSwitchOfBankOfGeorgia(t *testing.T) {
	t.Parallel()

	official := []string{getRates, "rates_list_official_history"}
	withCommercial := append([]string{getCommercial}, official...)

	tests := []struct {
		name         string
		on           bool
		key          string
		wantTools    []string
		wantWarnings int
		wantInfos    int
		wantRefusal  bool
	}{
		{name: "on, no key", on: true, wantTools: withCommercial, wantWarnings: 1},
		{name: "off, no key", wantTools: official, wantWarnings: 1, wantInfos: 1},
		{
			name:        "off, key",
			key:         "test-key",
			wantTools:   withCommercial,
			wantInfos:   1,
			wantRefusal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := commercialConfig(t, http.NotFoundHandler(), tt.key)
			if tt.on {
				cfg = withBOG(t, cfg, bogVendor(t))
			}

			sink := &syncBuffer{}
			session := serve(t, cfg, debugLogger(sink))

			if diff := cmp.Diff(tt.wantTools, toolNames(t, session)); diff != "" {
				t.Errorf("tools mismatch (-want +got):\n%s", diff)
			}

			warnings := warningsOf(sink)
			if got := len(recordsOf(sink, "WARN", tbcKey)); len(warnings) != tt.wantWarnings ||
				got != tt.wantWarnings {
				t.Errorf("warnings = %q, want %d that name %s", warnings, tt.wantWarnings, tbcKey)
			}

			if got := len(recordsOf(sink, "INFO", bogSwitch)); got != tt.wantInfos {
				t.Errorf("info records that name %s = %d, want %d", bogSwitch, got, tt.wantInfos)
			}

			if tt.on {
				assertDescriptionLists(t, session, bogCode)
			}

			if tt.wantRefusal {
				assertRefusesBankOfGeorgia(t, session)
			}
		})
	}
}

func assertDescriptionLists(t *testing.T, session *mcp.ClientSession, bank string) {
	t.Helper()

	list, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	for _, tool := range list.Tools {
		if tool.Name == getCommercial && !strings.Contains(tool.Description, bank) {
			t.Errorf("description = %q, want the bank %s", tool.Description, bank)
		}
	}
}

func assertRefusesBankOfGeorgia(t *testing.T, session *mcp.ClientSession) {
	t.Helper()

	result := call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar},
		argBanks:      []string{bogCode},
	})

	want := `banks: "bog" is not a bank of this server. Use one of: tbc.`
	if !result.IsError || text(result) != want {
		t.Errorf("result = %q, want the tool error %q", text(result), want)
	}
}

func recordsOf(sink *syncBuffer, level, needle string) []string {
	var records []string

	for line := range strings.Lines(sink.String()) {
		if strings.Contains(line, `"level":"`+level+`"`) && strings.Contains(line, needle) {
			records = append(records, line)
		}
	}

	return records
}
