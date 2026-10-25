package halyk_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/halyk"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	halykclient "github.com/abgeo/georgia-mcp/pkg/halyk"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/halyk/testdata"
	dollar   = "USD"
	franc    = "CHF"
	yen      = "JPY"
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
}

func page(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdata, "currency-ka.html")

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func row(code, buy, sell string) string {
	return `<tr><td><img alt="` + code + `"> ` + code + ` </td><td>` + buy + `</td><td>` +
		sell + `</td></tr>`
}

func dollarRow() string {
	return row(dollar, "2.5870", "2.6120")
}

func table(rows ...string) string {
	return `<html><body><div id="currentexchangerate"><table><thead><tr><th>ვალუტა</th>` +
		`<th>ყიდვა</th><th>გაყიდვა</th></tr></thead><tbody>` + strings.Join(rows, "") +
		`</tbody></table></div></body></html>`
}

func adapterFor(t *testing.T, status int, body string) *halyk.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return halyk.NewAdapter(halykclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

// HALRATE-SC-005: The adapter gives the price of one unit, with the trailing zeros
// dropped.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, page(t)).
		CommercialRates(t.Context(), []domain.CurrencyCode{"KZT", dollar, "EUR"})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	got := map[domain.CurrencyCode][2]string{}
	for _, rate := range result.Rates {
		got[rate.Currency] = [2]string{rate.Buy.String(), rate.Sell.String()}
	}

	want := map[domain.CurrencyCode][2]string{
		"EUR":  {"2.88", "2.94"},
		"KZT":  {"0.0054", "0.0059"},
		dollar: {"2.587", "2.612"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}

	if len(result.Missing) != 0 || !result.ReadAt.Equal(readAt()) {
		t.Errorf(
			"Missing = %v, ReadAt = %v, want none and %v",
			result.Missing,
			result.ReadAt,
			readAt(),
		)
	}
}

// HALRATE-SC-006: A code with no row, or with a price that is empty or zero, is missing.
func TestCommercialRatesReportAMissingCode(t *testing.T) {
	t.Parallel()

	for _, prices := range [][2]string{{"0.0000", "3.2500"}, {"", "3.2500"}} {
		t.Run(prices[0], func(t *testing.T) {
			t.Parallel()

			body := table(dollarRow(), row(franc, prices[0], prices[1]))

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

// HALRATE-SC-007: A broken row or a failure of the vendor wraps
// kernel.ErrSourceUnavailable.
func TestCommercialRatesReportABrokenSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "code not a code",
			status: http.StatusOK,
			body:   table(row("US$", "2.5870", "2.6120")),
		},
		{
			name:   "buy not a number",
			status: http.StatusOK,
			body:   table(row(dollar, "n/a", "2.6120")),
		},
		{
			name:   "buy below zero",
			status: http.StatusOK,
			body:   table(row(dollar, "-2.5870", "2.6120")),
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

// HALRATE-SC-009: The bank is Halyk Bank.
func TestBankIsHalyk(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, http.StatusOK, page(t)).Bank()
	want := domain.Bank{Code: "halyk", Name: "Halyk Bank", Kind: domain.KindBank}

	if got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
