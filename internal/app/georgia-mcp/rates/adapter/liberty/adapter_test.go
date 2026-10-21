package liberty_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/liberty"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	libertyclient "github.com/abgeo/georgia-mcp/pkg/libertyrates"
)

const (
	testdata = "../../../../../../pkg/libertyrates/testdata"
	dollar   = "USD"
	franc    = "CHF"
	yen      = "JPY"
	price    = `<span class="currency-rates__currency caps bold">`
)

func readAt() time.Time {
	return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
}

func page(t *testing.T) string {
	t.Helper()

	name := filepath.Join(testdata, "home-ka.html")

	body, err := os.ReadFile(name) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// replace returns the page with each old text as the new one. It fails when the page
// does not hold the old text.
func replace(t *testing.T, body string, pairs ...string) string {
	t.Helper()

	for i := 0; i < len(pairs); i += 2 {
		if !strings.Contains(body, pairs[i]) {
			t.Fatalf("the page holds no %q", pairs[i])
		}

		body = strings.ReplaceAll(body, pairs[i], pairs[i+1])
	}

	return body
}

func adapterFor(t *testing.T, status int, body string, calls *atomic.Int32) *liberty.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls != nil {
			calls.Add(1)
		}

		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return liberty.NewAdapter(libertyclient.New(libertyclient.WithBaseURL(server.URL)), readAt)
}

func formatted(rates []domain.CommercialRate) map[domain.CurrencyCode][2]string {
	out := make(map[domain.CurrencyCode][2]string, len(rates))
	for _, rate := range rates {
		out[rate.Currency] = [2]string{
			rate.Buy.StringFixed(-rate.Buy.Exponent()),
			rate.Sell.StringFixed(-rate.Sell.Exponent()),
		}
	}

	return out
}

// LIBRATE-SC-005: The adapter gives the commercial price of one unit, with the trailing
// zeros dropped, from one call.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	result, err := adapterFor(t, http.StatusOK, page(t), &calls).
		CommercialRates(t.Context(), []domain.CurrencyCode{"AMD", "EUR", dollar})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	want := map[domain.CurrencyCode][2]string{
		"AMD":  {"0.006222", "0.007573"},
		"EUR":  {"2.855", "2.971"},
		dollar: {"2.547", "2.647"},
	}
	if diff := cmp.Diff(want, formatted(result.Rates)); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}

	if len(result.Missing) != 0 || !result.ReadAt.Equal(readAt()) || calls.Load() != 1 {
		t.Errorf("Missing = %v, ReadAt = %v, calls = %d, want none, %v, 1",
			result.Missing, result.ReadAt, calls.Load(), readAt())
	}
}

// LIBRATE-SC-006: A code with no row, or with a commercial price of zero or none, is missing.
func TestCommercialRatesNameAMissingCode(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"a price of zero": {price + "3.0670</span>", price + "0.0000</span>"},
		"no price": {
			price + "3.0670</span>", "",
			price + "3.2280</span>", "",
		},
	}

	for name, pairs := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := adapterFor(t, http.StatusOK, replace(t, page(t), pairs...), nil).
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

// LIBRATE-SC-007: A unit that is not a power of 10 or names another code, a price that
// is not a number or is below zero, and a failed call fail the bank.
func TestCommercialRatesFailOnABrokenRow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		pairs  []string
	}{
		{name: "unit 3", status: http.StatusOK, pairs: []string{">1 USD<", ">3 USD<"}},
		{
			name:   "unit of another code",
			status: http.StatusOK,
			pairs:  []string{">1 USD<", ">1 EUR<"},
		},
		{
			name:   "not a number",
			status: http.StatusOK,
			pairs:  []string{price + "2.5470<", price + "n/a<"},
		},
		{
			name:   "below zero",
			status: http.StatusOK,
			pairs:  []string{price + "2.5470<", price + "-2.5470<"},
		},
		{name: "503", status: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := adapterFor(t, tt.status, replace(t, page(t), tt.pairs...), nil).
				CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("CommercialRates() error = %v, want ErrSourceUnavailable", err)
			}
		})
	}
}

func TestBank(t *testing.T) {
	t.Parallel()

	want := domain.Bank{Code: "liberty", Name: "Liberty Bank", Kind: domain.KindBank}
	if got := adapterFor(t, http.StatusOK, "", nil).Bank(); got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}

var _ app.CommercialRateSource = (*liberty.Adapter)(nil)
