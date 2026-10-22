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
	kursiTestdata = "../../../../pkg/kursi/testdata"
	kursiSwitch   = "GEOMCP_RATES_KURSI_ENABLED"
	kursiCode     = "kursi"
)

// kursiVendor serves the recorded answer.
func kursiVendor(t *testing.T) http.HandlerFunc {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(kursiTestdata, "currencies.json"))
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}
}

// withKursi turns Kursi.ge on and points it at the vendor.
func withKursi(t *testing.T, cfg rates.Config, vendor http.Handler) rates.Config {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	kursiURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Kursi = rates.KursiConfig{
		Enabled: true,
		BaseURL: *kursiURL,
		HTTP:    rates.KursiHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}

	return cfg
}

// KURSRATE-SC-010: The answer gives Kursi.ge, its read instant, its rate of USD, and
// GBP as missing. The description names it as an online exchange bureau.
func TestGetCommercialRatesOfKursi(t *testing.T) {
	t.Parallel()

	cfg := withKursi(t, commercialConfig(t, http.NotFoundHandler(), ""), kursiVendor(t))
	session := serve(t, cfg, debugLogger(&syncBuffer{}))

	assertDescriptionLists(t, session, "kursi (Kursi.ge, an online exchange bureau)")

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar, "GBP"},
	}))

	want := bogOutput{Banks: []bogBankOutput{{
		Bank:    kursiCode,
		Source:  "Kursi.ge",
		AsOf:    clockInstant,
		Rates:   []commercialRateOutput{{Currency: dollar, Buy: "2.594", Sell: "2.601"}},
		Missing: []string{"GBP"},
	}}}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// KURSRATE-SC-011: Off, Kursi.ge is not a bank of the server, and the server logs one
// info record that names the variable.
func TestTheSwitchOfKursi(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	cfg := withBOG(t, commercialConfig(t, http.NotFoundHandler(), ""), bogVendor(t))
	session := serve(t, cfg, debugLogger(sink))

	if got := len(infosOf(sink, kursiSwitch)); got != 1 {
		t.Errorf("info records that name %s = %d, want 1", kursiSwitch, got)
	}

	assertRefuses(
		t,
		session,
		kursiCode,
		`banks: "kursi" is not a bank of this server. Use one of: bog.`,
	)
}
