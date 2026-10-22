package rates_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
)

const (
	libertyTestdata = "../../../../pkg/libertyrates/testdata"
	libertySwitch   = "GEOMCP_RATES_LIBERTY_ENABLED"
	libertyCode     = "liberty"
)

// libertyVendor serves the recorded home page, and counts the calls.
func libertyVendor(t *testing.T, calls *atomic.Int32) http.HandlerFunc {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(libertyTestdata, "home-ka.html"))
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)

		_, _ = w.Write(body)
	}
}

// withLiberty turns Liberty on and points it at the vendor.
func withLiberty(t *testing.T, cfg rates.Config, vendor http.Handler) rates.Config {
	t.Helper()

	server := httptest.NewServer(vendor)
	t.Cleanup(server.Close)

	libertyURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Liberty = rates.LibertyConfig{
		Enabled: true,
		BaseURL: *libertyURL,
		HTTP:    rates.LibertyHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}

	return cfg
}

// LIBRATE-SC-009: The answer gives Liberty, its read instant, and the price of one
// unit, from one call.
func TestGetCommercialRatesOfLiberty(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	cfg := withLiberty(t, commercialConfig(t, http.NotFoundHandler(), ""), libertyVendor(t, &calls))
	session := serve(t, cfg, debugLogger(&syncBuffer{}))

	out := decode[bogOutput](t, call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar, "AMD"},
	}))

	want := bogOutput{Banks: []bogBankOutput{{
		Bank:   libertyCode,
		Source: "Liberty Bank",
		AsOf:   clockInstant,
		Rates: []commercialRateOutput{
			{Currency: "AMD", Buy: "0.006222", Sell: "0.007573"},
			{Currency: dollar, Buy: "2.547", Sell: "2.647"},
		},
	}}}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	if got := calls.Load(); got != 1 {
		t.Errorf("calls to Liberty = %d, want 1", got)
	}
}

// LIBRATE-SC-010: The switch of Liberty decides whether the server offers it. Off, the
// server logs one info record that names the variable.
func TestTheSwitchOfLiberty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		liberty   bool
		bog       bool
		wantInfos int
	}{
		{name: "on, other banks off", liberty: true},
		{name: "off, bog on", bog: true, wantInfos: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			cfg := commercialConfig(t, http.NotFoundHandler(), "")
			if tt.liberty {
				cfg = withLiberty(t, cfg, libertyVendor(t, &calls))
			}

			if tt.bog {
				cfg = withBOG(t, cfg, bogVendor(t))
			}

			sink := &syncBuffer{}
			session := serve(t, cfg, debugLogger(sink))

			if got := len(infosOf(sink, libertySwitch)); got != tt.wantInfos {
				t.Errorf(
					"info records that name %s = %d, want %d",
					libertySwitch,
					got,
					tt.wantInfos,
				)
			}

			if tt.liberty {
				assertDescriptionLists(t, session, libertyCode)

				return
			}

			assertRefuses(
				t,
				session,
				libertyCode,
				`banks: "liberty" is not a bank of this server. Use one of: bog.`,
			)
		})
	}
}

func assertRefuses(t *testing.T, session *mcp.ClientSession, bank, want string) {
	t.Helper()

	result := call(t, session, getCommercial, map[string]any{
		argCurrencies: []string{dollar},
		argBanks:      []string{bank},
	})

	if !result.IsError || text(result) != want {
		t.Errorf("result = %q, want the tool error %q", text(result), want)
	}
}
