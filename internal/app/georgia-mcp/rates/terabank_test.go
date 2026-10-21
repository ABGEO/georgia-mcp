package rates_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
)

const (
	terabankTestdata = "../../../../pkg/terabank/testdata"
	terabankSwitch   = "GEOMCP_RATES_TERABANK_ENABLED"
	terabankCode     = "terabank"
)

// terabankVendor serves the recorded answer.
func terabankVendor(t *testing.T) http.HandlerFunc {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(terabankTestdata, "cross-rates.json"))
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}
}

// withTerabank turns Terabank on and points it at the vendor.
func withTerabank(t *testing.T, cfg rates.Config, vendor http.Handler) rates.Config {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	terabankURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Terabank = rates.TerabankConfig{
		Enabled: true,
		BaseURL: *terabankURL,
		HTTP:    rates.TerabankHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}

	return cfg
}

// TERARATE-SC-009: The answer gives Terabank, its read instant, its rate of USD, and
// CHF as missing.
func TestGetCommercialRatesOfTerabank(t *testing.T) {
	t.Parallel()

	cfg := withTerabank(t, commercialConfig(t, http.NotFoundHandler(), ""), terabankVendor(t))
	session := serve(t, cfg, debugLogger(&syncBuffer{}))

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar, "CHF"},
	}))

	want := bogOutput{Banks: []bogBankOutput{{
		Bank:    terabankCode,
		Source:  "Terabank",
		AsOf:    clockInstant,
		Rates:   []commercialRateOutput{{Currency: dollar, Buy: "2.596", Sell: "2.601"}},
		Missing: []string{"CHF"},
	}}}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// TERARATE-SC-010: Off, Terabank is not a bank of the server, and the server logs one
// info record that names the variable.
func TestTheSwitchOfTerabank(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	cfg := withBOG(t, commercialConfig(t, http.NotFoundHandler(), ""), bogVendor(t))
	session := serve(t, cfg, debugLogger(sink))

	if got := len(recordsOf(sink, "INFO", terabankSwitch)); got != 1 {
		t.Errorf("info records that name %s = %d, want 1", terabankSwitch, got)
	}

	assertRefuses(t, session, terabankCode,
		`banks: "terabank" is not a bank of this server. Use one of: bog.`)
}
