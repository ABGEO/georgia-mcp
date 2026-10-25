package crystal_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/crystal"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	crystalclient "github.com/abgeo/georgia-mcp/pkg/crystal"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/crystal/testdata"
	dollar   = "USD"
	franc    = "CHF"
	yen      = "JPY"

	dollarItem = `{"SourceISO":"USD","TargetISO":"GEL","Multiplier":1,"BuyRate":2.595,"SellRate":2.599}`
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
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
	return `{"errorCode":"0","errorText":"","data":{"errorCode":"0","success":true,"data":[` +
		strings.Join(list, ",") + `]}}`
}

func adapterFor(t *testing.T, status int, body string) *crystal.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return crystal.NewAdapter(crystalclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

func pricesOf(t *testing.T, codes ...domain.CurrencyCode) map[domain.CurrencyCode][2]string {
	t.Helper()

	result, err := adapterFor(t, http.StatusOK, recorded(t, "web-rate-cash.json")).
		CommercialRates(t.Context(), codes)
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

// CRYSRATE-SC-005: The adapter gives the price of one unit, with the trailing zeros
// dropped, and no cross rate.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	want := map[domain.CurrencyCode][2]string{
		"EUR":  {"2.905", "2.914"},
		dollar: {"2.595", "2.599"},
	}
	if diff := cmp.Diff(want, pricesOf(t, "EUR", dollar)); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}
}

// CRYSRATE-SC-006: The adapter divides a price of 10 or 1000 units by the multiplier.
func TestCommercialRatesDivideByTheMultiplier(t *testing.T) {
	t.Parallel()

	want := map[domain.CurrencyCode][2]string{
		"AMD": {"0.0067", "0.007"},
		"ILS": {"0.35", "0.8"},
	}
	if diff := cmp.Diff(want, pricesOf(t, "AMD", "ILS")); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}
}

// CRYSRATE-SC-007: A code with no item, or with a price that is absent or zero, is
// missing.
func TestCommercialRatesReportAMissingCode(t *testing.T) {
	t.Parallel()

	for _, item := range []string{
		`{"SourceISO":"CHF","TargetISO":"GEL","Multiplier":1,"BuyRate":0,"SellRate":3.14}`,
		`{"SourceISO":"CHF","TargetISO":"GEL","Multiplier":1,"BuyRate":3.08}`,
	} {
		t.Run(item, func(t *testing.T) {
			t.Parallel()

			result, err := adapterFor(t, http.StatusOK, items(dollarItem, item)).
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

// CRYSRATE-SC-008: A broken answer or a failure of the vendor wraps
// kernel.ErrSourceUnavailable.
func TestCommercialRatesReportABrokenSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "not succeeded", status: http.StatusOK, body: recorded(t, "not-succeeded.json")},
		{name: "status 503", status: http.StatusServiceUnavailable, body: "down"},
		{
			name:   "buy not a number",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"BuyRate":2.595`, `"BuyRate":"n/a"`, 1)),
		},
		{
			name:   "buy below zero",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"BuyRate":2.595`, `"BuyRate":-1`, 1)),
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

// CRYSRATE-SC-010: The bank is Crystal, a microfinance organization.
func TestBankIsCrystal(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, http.StatusOK, recorded(t, "web-rate-cash.json")).Bank()
	want := domain.Bank{Code: "crystal", Name: "Crystal", Kind: domain.KindMicrofinance}

	if got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
