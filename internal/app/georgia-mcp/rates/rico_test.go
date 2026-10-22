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
	ricoTestdata = "../../../../pkg/rico/testdata"
	ricoSwitch   = "GEOMCP_RATES_RICO_ENABLED"
	ricoCode     = "rico"
)

// ricoVendor serves the recorded home page.
func ricoVendor(t *testing.T) http.HandlerFunc {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(ricoTestdata, "home-ka.html"))
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}
}

// withRico turns Rico Credit on and points it at the vendor.
func withRico(t *testing.T, cfg rates.Config, vendor http.Handler) rates.Config {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	ricoURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Rico = rates.RicoConfig{
		Enabled: true,
		BaseURL: *ricoURL,
		HTTP:    rates.RicoHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}

	return cfg
}

// RICORATE-SC-010: The description names Rico Credit as a microfinance organization,
// and the answer gives its rate of USD.
func TestGetCommercialRatesOfRicoCredit(t *testing.T) {
	t.Parallel()

	cfg := withRico(t, commercialConfig(t, http.NotFoundHandler(), ""), ricoVendor(t))
	session := serve(t, cfg, debugLogger(&syncBuffer{}))

	assertDescriptionLists(t, session, "rico (Rico Credit, a microfinance organization)")

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar},
	}))

	want := bogOutput{Banks: []bogBankOutput{{
		Bank:   ricoCode,
		Source: "Rico Credit",
		AsOf:   clockInstant,
		Rates:  []commercialRateOutput{{Currency: dollar, Buy: "2.596", Sell: "2.6"}},
	}}}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// RICORATE-SC-011: Off, Rico Credit is not a bank of the server, and the server logs
// one info record that names the variable.
func TestTheSwitchOfRicoCredit(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	cfg := withBOG(t, commercialConfig(t, http.NotFoundHandler(), ""), bogVendor(t))
	session := serve(t, cfg, debugLogger(sink))

	if got := len(infosOf(sink, ricoSwitch)); got != 1 {
		t.Errorf("info records that name %s = %d, want 1", ricoSwitch, got)
	}

	assertRefuses(
		t,
		session,
		ricoCode,
		`banks: "rico" is not a bank of this server. Use one of: bog.`,
	)
}
