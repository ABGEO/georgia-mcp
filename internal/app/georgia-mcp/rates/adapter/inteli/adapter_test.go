package inteli_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/inteli"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	inteliclient "github.com/abgeo/georgia-mcp/pkg/inteli"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/inteli/testdata"
	lari     = "GEL"
	dollar   = "USD"
	pound    = "GBP"
	franc    = "CHF"

	poundSell = "3.496"
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
}

func page(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdata, "main-page-geo.html")

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func row(base, currency, buy, sell string) string {
	return `<tr><td>` + base + `</td><td>` + currency + `</td><td>` + buy + `</td><td>` +
		sell + `</td></tr>`
}

func dollarRow() string {
	return row(lari, dollar, "2.595", "2.598")
}

func table(rows ...string) string {
	return `<html><body><table><thead><tr><th colspan="5">ვალუტის სტანდარტული კურსები</th>` +
		`</tr></thead><tbody>` + strings.Join(rows, "") + `</tbody></table></body></html>`
}

func adapterFor(t *testing.T, status int, body string) *inteli.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return inteli.NewAdapter(inteliclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

// INTLRATE-SC-005: The adapter gives the price of one unit, with the trailing zeros
// dropped, and no cross rate.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, page(t)).
		CommercialRates(t.Context(), []domain.CurrencyCode{"EUR", pound, "TRY", dollar})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	got := map[domain.CurrencyCode][2]string{}
	for _, rate := range result.Rates {
		got[rate.Currency] = [2]string{rate.Buy.String(), rate.Sell.String()}
	}

	want := map[domain.CurrencyCode][2]string{
		"EUR":  {"2.903", "2.915"},
		pound:  {"3.457", poundSell},
		"TRY":  {"0.05", "0.056"},
		dollar: {"2.595", "2.598"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}

	if len(result.Rates) != 4 || len(result.Missing) != 0 || !result.ReadAt.Equal(readAt()) {
		t.Errorf(
			"Rates = %v, Missing = %v, ReadAt = %v, want 4 rates, none missing, and %v",
			result.Rates,
			result.Missing,
			result.ReadAt,
			readAt(),
		)
	}
}

// INTLRATE-SC-006: A code with no row, or with a price that is empty or zero, is missing.
func TestCommercialRatesReportAMissingCode(t *testing.T) {
	t.Parallel()

	for _, prices := range [][2]string{{"0.000", poundSell}, {"", poundSell}} {
		t.Run(prices[0], func(t *testing.T) {
			t.Parallel()

			body := table(dollarRow(), row(lari, pound, prices[0], prices[1]))

			result, err := adapterFor(t, http.StatusOK, body).
				CommercialRates(t.Context(), []domain.CurrencyCode{franc, pound, dollar})
			if err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %v, want the rate of USD", result.Rates)
			}

			if diff := cmp.Diff([]domain.CurrencyCode{franc, pound}, result.Missing); diff != "" {
				t.Errorf("Missing mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// INTLRATE-SC-007: A buy price above its sell price in any row against the lari fails
// the bank. Equal prices and a cross rate pass.
func TestCommercialRatesCheckTheOrderOfThePrices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		second string
		fails  bool
	}{
		{name: "lower buy", second: row(lari, pound, "3.457", poundSell)},
		{name: "equal prices", second: row(lari, pound, "3.457", "3.457")},
		{name: "higher buy", second: row(lari, pound, poundSell, "3.457"), fails: true},
		{name: "cross rate", second: row("EUR", dollar, "1.123", "1.119")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := adapterFor(t, http.StatusOK, table(dollarRow(), tt.second)).
				CommercialRates(t.Context(), []domain.CurrencyCode{dollar})

			if tt.fails {
				if !errors.Is(err, kernel.ErrSourceUnavailable) {
					t.Errorf("CommercialRates() error = %v, want kernel.ErrSourceUnavailable", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %v, want the rate of USD", result.Rates)
			}
		})
	}
}

// INTLRATE-SC-008: A broken row or a failure of the vendor wraps
// kernel.ErrSourceUnavailable.
func TestCommercialRatesReportABrokenSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "code of two letters",
			status: http.StatusOK,
			body:   table(row(lari, "US", "2.595", "2.598")),
		},
		{
			name:   "buy not a number",
			status: http.StatusOK,
			body:   table(row(lari, dollar, "n/a", "2.598")),
		},
		{
			name:   "buy below zero",
			status: http.StatusOK,
			body:   table(row(lari, dollar, "-2.595", "2.598")),
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

// INTLRATE-SC-010: The bank is Inteli Express, a microfinance organization.
func TestBankIsInteliExpress(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, http.StatusOK, page(t)).Bank()
	want := domain.Bank{Code: "inteli", Name: "Inteli Express", Kind: domain.KindMicrofinance}

	if got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
