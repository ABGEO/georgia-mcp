package terabank_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/terabank"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	terabankclient "github.com/abgeo/georgia-mcp/pkg/terabank"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/terabank/testdata"
	dollar   = "USD"
	franc    = "CHF"
	yen      = "JPY"
	usdItem  = `{"iso":"USD","teraCrossRateBuy":2.5960,"teraCrossRateSell":2.6010}`
)

func readAt() time.Time {
	return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
}

func recorded(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(testdata, name)

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func succeeded(items string) string {
	return `{"data":[` + items + `],"succeeded":true,"message":""}`
}

func adapterFor(t *testing.T, status int, body string) *terabank.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return terabank.NewAdapter(terabankclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

// TERARATE-SC-005: The adapter gives the price of each code, with the trailing zeros dropped.
func TestCommercialRatesGiveThePublishedPrice(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, recorded(t, "cross-rates.json")).
		CommercialRates(t.Context(), []domain.CurrencyCode{"EUR", "GBP", dollar})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	got := make(map[domain.CurrencyCode][2]string, len(result.Rates))
	for _, rate := range result.Rates {
		got[rate.Currency] = [2]string{
			rate.Buy.StringFixed(-rate.Buy.Exponent()),
			rate.Sell.StringFixed(-rate.Sell.Exponent()),
		}
	}

	want := map[domain.CurrencyCode][2]string{
		"EUR":  {"2.905", "2.93"},
		"GBP":  {"3.376", "3.56"},
		dollar: {"2.596", "2.601"},
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

// TERARATE-SC-006: A code with no item, or with a price of zero or none, is missing.
func TestCommercialRatesNameAMissingCode(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"a price of zero": `{"iso":"CHF","teraCrossRateBuy":0,"teraCrossRateSell":3.2}`,
		"no sell price":   `{"iso":"CHF","teraCrossRateBuy":3.1}`,
	}

	for name, item := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := adapterFor(t, http.StatusOK, succeeded(usdItem+","+item)).
				CommercialRates(t.Context(), []domain.CurrencyCode{franc, yen, dollar})
			if err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %+v, want USD only", result.Rates)
			}

			if diff := cmp.Diff([]domain.CurrencyCode{franc, yen}, result.Missing); diff != "" {
				t.Errorf("Missing mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TERARATE-SC-007: An answer that did not succeed, a failed call, and a price that is
// not a number or is below zero fail the bank.
func TestCommercialRatesFailOnABrokenAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "not succeeded", status: http.StatusOK, body: recorded(t, "not-succeeded.json")},
		{name: "500", status: http.StatusInternalServerError},
		{
			name:   "not a number",
			status: http.StatusOK,
			body:   succeeded(`{"iso":"USD","teraCrossRateBuy":"n/a","teraCrossRateSell":2.601}`),
		},
		{
			name:   "below zero",
			status: http.StatusOK,
			body:   succeeded(`{"iso":"USD","teraCrossRateBuy":-1,"teraCrossRateSell":2.601}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := adapterFor(t, tt.status, tt.body).
				CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("CommercialRates() error = %v, want ErrSourceUnavailable", err)
			}
		})
	}
}

func TestBank(t *testing.T) {
	t.Parallel()

	want := domain.Bank{Code: "terabank", Name: "Terabank", Kind: domain.KindBank}
	if got := adapterFor(t, http.StatusOK, "").Bank(); got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
