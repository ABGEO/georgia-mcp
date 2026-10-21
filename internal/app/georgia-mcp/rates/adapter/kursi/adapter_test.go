package kursi_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/kursi"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	kursiclient "github.com/abgeo/georgia-mcp/pkg/kursi"
)

const (
	testdata = "../../../../../../pkg/kursi/testdata"
	dollar   = "USD"
	franc    = "CHF"
	usdPair  = `{"baseCurrencyCode":"GEL","secondaryCurrencyCode":"USD","buyRate":2.5940,"sellRate":2.6010}`
)

func readAt() time.Time {
	return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
}

func recorded(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdata, "currencies.json")

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func pairs(items ...string) string {
	return "[" + strings.Join(items, ",") + "]"
}

func adapterFor(t *testing.T, status int, body string) *kursi.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return kursi.NewAdapter(kursiclient.New(kursiclient.WithBaseURL(server.URL)), readAt)
}

// KURSRATE-SC-005: The adapter reads the pairs against the lari, with the trailing
// zeros dropped. The rate of EUR is not the price of the pair of USD and EUR.
func TestCommercialRatesReadThePairsOfTheLari(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, recorded(t)).
		CommercialRates(t.Context(), []domain.CurrencyCode{"CNY", "EUR", dollar})
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
		"CNY":  {"0.3858", "0.3925"},
		"EUR":  {"2.911", "2.926"},
		dollar: {"2.594", "2.601"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}

	if len(result.Missing) != 0 || !result.ReadAt.Equal(readAt()) {
		t.Errorf("Missing = %v, ReadAt = %v, want none and %v",
			result.Missing, result.ReadAt, readAt())
	}
}

// KURSRATE-SC-006: A code with no pair against the lari, or with a price of zero, is missing.
func TestCommercialRatesNameAMissingCode(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"a price of zero": `{"baseCurrencyCode":"GEL","secondaryCurrencyCode":"CHF","buyRate":0,"sellRate":3.2}`,
		"no pair of the lari": `{"baseCurrencyCode":"USD","secondaryCurrencyCode":"CHF",` +
			`"buyRate":0.8,"sellRate":0.9}`,
	}

	for name, pair := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := adapterFor(t, http.StatusOK, pairs(usdPair, pair)).
				CommercialRates(t.Context(), []domain.CurrencyCode{franc, "GBP", dollar})
			if err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %+v, want USD only", result.Rates)
			}

			if diff := cmp.Diff([]domain.CurrencyCode{franc, "GBP"}, result.Missing); diff != "" {
				t.Errorf("Missing mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// KURSRATE-SC-007: A failed call, a price that is not a number or is below zero, and
// two pairs of the lari for one code fail the bank.
func TestCommercialRatesFailOnABrokenAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "500", status: http.StatusInternalServerError},
		{
			name:   "not a number",
			status: http.StatusOK,
			body:   strings.Replace(usdPair, `"buyRate":2.5940`, `"buyRate":"n/a"`, 1),
		},
		{
			name:   "below zero",
			status: http.StatusOK,
			body:   pairs(strings.Replace(usdPair, `"buyRate":2.5940`, `"buyRate":-1`, 1)),
		},
		{name: "two pairs", status: http.StatusOK, body: pairs(usdPair, usdPair)},
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

	want := domain.Bank{Code: "kursi", Name: "Kursi.ge", Kind: domain.KindExchangeBureau}
	if got := adapterFor(t, http.StatusOK, "").Bank(); got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
