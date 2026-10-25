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
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	franc = "CHF"

	scappCode     = "scapp"
	halykCode     = "halyk"
	basisbankCode = "basisbank"
	credoCode     = "credo"

	scappSwitch     = "GEOMCP_RATES_SCAPP_ENABLED"
	halykSwitch     = "GEOMCP_RATES_HALYK_ENABLED"
	basisbankSwitch = "GEOMCP_RATES_BASISBANK_ENABLED"
	credoSwitch     = "GEOMCP_RATES_CREDO_ENABLED" //nolint:gosec // A name, not a key.

	// credoDate is clockInstant in UTC, as Credo Bank reads it.
	credoDate = "2026-10-18T10:00:00.000Z"
)

// bankVendor serves a recorded answer of a vendor. The answer is a file of the
// testdata of the client.
func bankVendor(t *testing.T, client, name string) http.HandlerFunc {
	t.Helper()

	path := filepath.Join("../../../../pkg", client, "testdata", name)

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}
}

// vendorAt serves the vendor and returns a configuration of a bank that is on and
// points at it.
func vendorAt(t *testing.T, vendor http.Handler) vendors.Config {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return vendors.Config{
		Enabled: true,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
		BaseURL: *address,
	}
}

// assertOneBank calls the tool with USD and CHF, and compares the answer with the one
// bank that is on.
func assertOneBank(t *testing.T, cfg rates.Config, want bogBankOutput) {
	t.Helper()

	session := serve(t, cfg, debugLogger(&syncBuffer{}))

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar, franc},
	}))

	if diff := cmp.Diff(bogOutput{Banks: []bogBankOutput{want}}, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// assertSwitchedOff serves Bank of Georgia only, and checks that the bank is refused and
// that one info record names its switch.
func assertSwitchedOff(t *testing.T, code, variable string) {
	t.Helper()

	sink := &syncBuffer{}
	cfg := withBOG(t, commercialConfig(t, http.NotFoundHandler(), ""), bogVendor(t))
	session := serve(t, cfg, debugLogger(sink))

	if got := len(infosOf(sink, variable)); got != 1 {
		t.Errorf("info records that name %s = %d, want 1", variable, got)
	}

	assertRefuses(t, session, code,
		`banks: "`+code+`" is not a bank of this server. Use one of: bog.`)
}

// SCAPRATE-SC-010: The description names Scapp as a microfinance organization, and the
// answer gives its read instant, its rate of USD, and CHF as missing.
func TestGetCommercialRatesOfScapp(t *testing.T) {
	t.Parallel()

	cfg := commercialConfig(t, http.NotFoundHandler(), "")
	cfg.Scapp = vendorAt(t, bankVendor(t, "scapp", "currency-ka.html"))

	assertDescriptionLists(
		t,
		serve(t, cfg, debugLogger(&syncBuffer{})),
		"scapp (Scapp, a microfinance organization)",
	)

	assertOneBank(t, cfg, bogBankOutput{
		Bank:    scappCode,
		Source:  "Scapp",
		AsOf:    clockInstant,
		Rates:   []commercialRateOutput{{Currency: dollar, Buy: "2.595", Sell: "2.599"}},
		Missing: []string{franc},
	})
}

// SCAPRATE-SC-011: Off, Scapp is not a bank of the server, and the server logs one info
// record that names the variable.
func TestTheSwitchOfScapp(t *testing.T) {
	t.Parallel()

	assertSwitchedOff(t, scappCode, scappSwitch)
}

// HALRATE-SC-009: The answer gives Halyk Bank, its read instant, its rate of USD, and CHF
// as missing.
func TestGetCommercialRatesOfHalyk(t *testing.T) {
	t.Parallel()

	cfg := commercialConfig(t, http.NotFoundHandler(), "")
	cfg.Halyk = vendorAt(t, bankVendor(t, "halyk", "currency-ka.html"))

	assertOneBank(t, cfg, bogBankOutput{
		Bank:    halykCode,
		Source:  "Halyk Bank",
		AsOf:    clockInstant,
		Rates:   []commercialRateOutput{{Currency: dollar, Buy: "2.587", Sell: "2.612"}},
		Missing: []string{franc},
	})
}

// HALRATE-SC-010: Off, Halyk Bank is not a bank of the server, and the server logs one
// info record that names the variable.
func TestTheSwitchOfHalyk(t *testing.T) {
	t.Parallel()

	assertSwitchedOff(t, halykCode, halykSwitch)
}

// BASRATE-SC-010: The answer gives Basisbank, its read instant, its rate of USD, and CHF
// as missing.
func TestGetCommercialRatesOfBasisbank(t *testing.T) {
	t.Parallel()

	cfg := commercialConfig(t, http.NotFoundHandler(), "")
	cfg.Basisbank = vendorAt(t, bankVendor(t, "basisbank", "website-xrates.json"))

	assertOneBank(t, cfg, bogBankOutput{
		Bank:    basisbankCode,
		Source:  "Basisbank",
		AsOf:    clockInstant,
		Rates:   []commercialRateOutput{{Currency: dollar, Buy: "2.56", Sell: "2.63"}},
		Missing: []string{franc},
	})
}

// BASRATE-SC-011: Off, Basisbank is not a bank of the server, and the server logs one
// info record that names the variable.
func TestTheSwitchOfBasisbank(t *testing.T) {
	t.Parallel()

	assertSwitchedOff(t, basisbankCode, basisbankSwitch)
}

// CREDRATE-SC-011: The vendor receives the instant of the clock and the key, and the
// answer gives Credo Bank, its read instant, its rate of USD, and CHF as missing.
func TestGetCommercialRatesOfCredo(t *testing.T) {
	t.Parallel()

	dates := make(chan string, 1)
	keys := make(chan string, 1)
	recorded := bankVendor(t, "credo", "current-rates.json")

	cfg := commercialConfig(t, http.NotFoundHandler(), "")
	cfg.Credo = rates.CredoConfig{
		Config: vendorAt(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dates <- r.URL.Query().Get("date")

			keys <- r.Header.Get("X-Api-Key")

			recorded(w, r)
		})),
		APIKey: "credo-key",
	}

	assertOneBank(t, cfg, bogBankOutput{
		Bank:    credoCode,
		Source:  "Credo Bank",
		AsOf:    clockInstant,
		Rates:   []commercialRateOutput{{Currency: dollar, Buy: "2.593", Sell: "2.607"}},
		Missing: []string{franc},
	})

	if date, key := <-dates, <-keys; date != credoDate || key != "credo-key" {
		t.Errorf("vendor received date %q and key %q, want %q and credo-key", date, key, credoDate)
	}
}

// CREDRATE-SC-012: Off, Credo Bank is not a bank of the server, and the server logs one
// info record that names the variable.
func TestTheSwitchOfCredo(t *testing.T) {
	t.Parallel()

	assertSwitchedOff(t, credoCode, credoSwitch)
}
