package mjc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/mjc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	mjcclient "github.com/abgeo/georgia-mcp/pkg/mjc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/mjc/testdata"
	dollar   = "USD"
	franc    = "CHF"
	yen      = "JPY"
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
}

func page(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdata, "rates.html")

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func row(unit, buy, sell string) string {
	return `<tr><td><pre>` + unit + `</pre></td><td><pre>` + buy + `</pre></td><td><pre>` +
		sell + `</pre></td><td><pre>N/A</pre></td></tr>`
}

func dollarRow() string {
	return row("1 USD", "2.59500", "2.59800")
}

func table(rows ...string) string {
	return `<html><body><table><thead><tr><th>ვალუტა</th><th>ყიდვა</th>` +
		`<th>გაყიდვა</th><th>ეროვნული</th></tr></thead><tbody>` + strings.Join(rows, "") +
		`</tbody></table></body></html>`
}

func adapterFor(t *testing.T, status int, body string) *mjc.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return mjc.NewAdapter(mjcclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

func pricesOf(
	t *testing.T,
	body string,
	codes ...domain.CurrencyCode,
) map[domain.CurrencyCode][2]string {
	t.Helper()

	result, err := adapterFor(t, http.StatusOK, body).CommercialRates(t.Context(), codes)
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	if len(result.Missing) != 0 || !result.ReadAt.Equal(readAt()) {
		t.Errorf(
			"Missing = %v, ReadAt = %v, want none and %v",
			result.Missing,
			result.ReadAt,
			readAt(),
		)
	}

	got := map[domain.CurrencyCode][2]string{}
	for _, rate := range result.Rates {
		got[rate.Currency] = [2]string{rate.Buy.String(), rate.Sell.String()}
	}

	return got
}

// MJCRATE-SC-005: The adapter gives the price of one unit, with the trailing zeros
// dropped, also for a row with no official rate.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	got := pricesOf(t, page(t), "EUR", "SAR", dollar)

	want := map[domain.CurrencyCode][2]string{
		"EUR":  {"2.907", "2.912"},
		"SAR":  {"0.669", "0.685"},
		dollar: {"2.595", "2.598"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}
}

// MJCRATE-SC-006: The adapter divides a price of 10 or 1000 units by the unit.
func TestCommercialRatesDivideByTheUnit(t *testing.T) {
	t.Parallel()

	got := pricesOf(t, page(t), "AMD", "PLN")

	want := map[domain.CurrencyCode][2]string{
		"AMD": {"0.007", "0.0072"},
		"PLN": {"0.634", "0.654"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}
}

// MJCRATE-SC-007: A code with no row, or with a price that is empty or zero, is missing.
func TestCommercialRatesReportAMissingCode(t *testing.T) {
	t.Parallel()

	for _, prices := range [][2]string{{"0.00000", "3.14300"}, {"", "3.14300"}} {
		t.Run(prices[0], func(t *testing.T) {
			t.Parallel()

			body := table(dollarRow(), row("1 CHF", prices[0], prices[1]))

			result, err := adapterFor(t, http.StatusOK, body).
				CommercialRates(t.Context(), []domain.CurrencyCode{franc, yen, dollar})
			if err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %v, want the rate of USD", result.Rates)
			}

			if diff := cmp.Diff([]domain.CurrencyCode{franc, yen}, result.Missing); diff != "" {
				t.Errorf("Missing mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// MJCRATE-SC-008: A broken row or a failure of the vendor wraps
// kernel.ErrSourceUnavailable.
func TestCommercialRatesReportABrokenSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "unit of 3", status: http.StatusOK, body: table(row("3 USD", "2.59500", "2.59800"))},
		{
			name:   "unit with no count",
			status: http.StatusOK,
			body:   table(row("USD", "2.59500", "2.59800")),
		},
		{
			name:   "buy not a number",
			status: http.StatusOK,
			body:   table(row("1 USD", "N/A", "2.59800")),
		},
		{
			name:   "buy below zero",
			status: http.StatusOK,
			body:   table(row("1 USD", "-2.59500", "2.59800")),
		},
		{name: "two rows of USD", status: http.StatusOK, body: table(dollarRow(), dollarRow())},
		{name: "status 503", status: http.StatusServiceUnavailable, body: "down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := adapterFor(t, tt.status, tt.body).
				CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("CommercialRates() error = %v, want kernel.ErrSourceUnavailable", err)
			}
		})
	}
}

// MJCRATE-SC-010: The bank is MJC, a microfinance organization.
func TestBankIsMJC(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, http.StatusOK, page(t)).Bank()
	want := domain.Bank{Code: "mjc", Name: "MJC", Kind: domain.KindMicrofinance}

	if got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
