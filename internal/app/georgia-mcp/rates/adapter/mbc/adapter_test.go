package mbc_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/mbc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	mbcclient "github.com/abgeo/georgia-mcp/pkg/mbc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/mbc/testdata"
	dollar   = "USD"
	euro     = "EUR"

	dollarItem = `{"FromCcy":"USD","ToCcy":"GEL","Multiplier":1,"Buy":2.5950,"Sell":2.5990}`
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 12, 25, 27, 0, time.UTC)
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

func items(list ...string) string {
	return `{"FXRates":[` + strings.Join(list, ",") + `],"timestamp":1}`
}

func adapterFor(t *testing.T, status int, body string) *mbc.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return mbc.NewAdapter(mbcclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

// MBCRATE-SC-005: The adapter gives the price of one unit, with the trailing zeros
// dropped, and the instant of its clock.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, recorded(t, "commercial.json")).
		CommercialRates(t.Context(), []domain.CurrencyCode{dollar, euro})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	got := map[domain.CurrencyCode][2]string{}
	for _, rate := range result.Rates {
		got[rate.Currency] = [2]string{rate.Buy.String(), rate.Sell.String()}
	}

	want := map[domain.CurrencyCode][2]string{
		euro:   {"2.907", "2.914"},
		dollar: {"2.595", "2.599"},
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

// MBCRATE-SC-006: A code that the page hides is missing, though the method lists it.
func TestCommercialRatesKeepTheCodesOfThePage(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, recorded(t, "commercial.json")).
		CommercialRates(t.Context(), []domain.CurrencyCode{"GBP", "TRY", dollar})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
		t.Errorf("Rates = %v, want the rate of USD", result.Rates)
	}

	if diff := cmp.Diff([]domain.CurrencyCode{"GBP", "TRY"}, result.Missing); diff != "" {
		t.Errorf("Missing mismatch (-want +got):\n%s", diff)
	}
}

// MBCRATE-SC-007: The adapter divides a price of 10 units by 10.
func TestCommercialRatesDivideByTheMultiplier(t *testing.T) {
	t.Parallel()

	body := items(
		dollarItem,
		`{"FromCcy":"EUR","ToCcy":"GEL","Multiplier":10,"Buy":29.0700,"Sell":29.1400}`,
	)

	result, err := adapterFor(t, http.StatusOK, body).
		CommercialRates(t.Context(), []domain.CurrencyCode{euro})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	if len(result.Rates) != 1 {
		t.Fatalf("Rates = %v, want one rate", result.Rates)
	}

	rate := result.Rates[0]
	if rate.Buy.String() != "2.907" || rate.Sell.String() != "2.914" {
		t.Errorf("EUR = %s / %s, want 2.907 / 2.914", rate.Buy, rate.Sell)
	}
}

// MBCRATE-SC-008: A code with a price that is absent or zero, or with no item against
// GEL, is missing.
func TestCommercialRatesReportAMissingCode(t *testing.T) {
	t.Parallel()

	for _, item := range []string{
		`{"FromCcy":"EUR","ToCcy":"GEL","Multiplier":1,"Buy":0,"Sell":2.914}`,
		`{"FromCcy":"EUR","ToCcy":"GEL","Multiplier":1,"Buy":2.907}`,
		`{"FromCcy":"EUR","ToCcy":"USD","Multiplier":1,"Buy":1.11,"Sell":1.13}`,
	} {
		t.Run(item, func(t *testing.T) {
			t.Parallel()

			result, err := adapterFor(t, http.StatusOK, items(dollarItem, item)).
				CommercialRates(t.Context(), []domain.CurrencyCode{euro, dollar})
			if err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %v, want the rate of USD", result.Rates)
			}

			if diff := cmp.Diff([]domain.CurrencyCode{euro}, result.Missing); diff != "" {
				t.Errorf("Missing mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// MBCRATE-SC-009: A broken answer or a failure of the vendor wraps
// kernel.ErrSourceUnavailable.
func TestCommercialRatesReportABrokenSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "no rates", status: http.StatusOK, body: recorded(t, "no-rates.json")},
		{name: "status 503", status: http.StatusServiceUnavailable, body: "down"},
		{
			name:   "buy not a number",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"Buy":2.5950`, `"Buy":"n/a"`, 1)),
		},
		{
			name:   "buy below zero",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"Buy":2.5950`, `"Buy":-1`, 1)),
		},
		{
			name:   "multiplier of 3",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"Multiplier":1`, `"Multiplier":3`, 1)),
		},
		{name: "two items of USD", status: http.StatusOK, body: items(dollarItem, dollarItem)},
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

// MBCRATE-SC-011: The bank is MBC, a microfinance organization.
func TestBankIsMBC(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, http.StatusOK, recorded(t, "commercial.json")).Bank()
	want := domain.Bank{Code: "mbc", Name: "MBC", Kind: domain.KindMicrofinance}

	if got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
