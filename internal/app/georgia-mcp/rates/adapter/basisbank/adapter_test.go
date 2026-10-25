package basisbank_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/basisbank"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	basisclient "github.com/abgeo/georgia-mcp/pkg/basisbank"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/basisbank/testdata"
	dollar   = "USD"
	franc    = "CHF"
	yen      = "JPY"

	dollarSell = "2.6300"
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
}

func recorded(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdata, "website-xrates.json")

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// day is a date of the answer with its buy and sell prices.
type day struct {
	date      string
	buy, sell map[string]string
}

// answer encodes the days as the vendor does: the rates of each day are a JSON text.
func answer(t *testing.T, days ...day) string {
	t.Helper()

	wire := make([]map[string]string, 0, len(days))

	for _, item := range days {
		rates, err := json.Marshal(map[string]map[string]string{
			"kurs": {}, "kursBuy": item.buy, "kursSell": item.sell,
		})
		if err != nil {
			t.Fatal(err)
		}

		wire = append(wire, map[string]string{"date": item.date, "xrates": string(rates)})
	}

	body, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func dollarDay(date, buy string) day {
	return day{
		date: date,
		buy:  map[string]string{dollar: buy},
		sell: map[string]string{dollar: dollarSell},
	}
}

func adapterFor(t *testing.T, status int, body string) *basisbank.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return basisbank.NewAdapter(basisclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

func dollarBuy(t *testing.T, body string) (string, error) {
	t.Helper()

	result, err := adapterFor(t, http.StatusOK, body).
		CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
	if err != nil {
		return "", err
	}

	if len(result.Rates) != 1 {
		t.Fatalf("Rates = %v, want the rate of USD", result.Rates)
	}

	return result.Rates[0].Buy.String(), nil
}

// BASRATE-SC-005: The adapter gives the price of one unit, with the trailing zeros
// dropped.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, recorded(t)).
		CommercialRates(t.Context(), []domain.CurrencyCode{dollar, "EUR", "CNY"})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	got := map[domain.CurrencyCode][2]string{}
	for _, rate := range result.Rates {
		got[rate.Currency] = [2]string{rate.Buy.String(), rate.Sell.String()}
	}

	want := map[domain.CurrencyCode][2]string{
		"CNY":  {"0.37", "0.41"},
		"EUR":  {"2.816", "2.998"},
		dollar: {"2.56", "2.63"},
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

// BASRATE-SC-006: The adapter reads the newest day in either order.
func TestCommercialRatesReadTheNewestDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		days []day
	}{
		{
			name: "newest first",
			days: []day{dollarDay("2026-10-25", "2.5600"), dollarDay("2026-10-24", "2.5500")},
		},
		{
			name: "newest last",
			days: []day{dollarDay("2026-10-24", "2.5500"), dollarDay("2026-10-25", "2.5600")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buy, err := dollarBuy(t, answer(t, tt.days...))
			if err != nil || buy != "2.56" {
				t.Errorf("buy of USD = %q, %v, want 2.56", buy, err)
			}
		})
	}
}

// BASRATE-SC-006: No day, a date in another format, or a date twice fails the bank.
func TestCommercialRatesReportABrokenDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "no day", body: "[]"},
		{name: "date of another format", body: answer(t, dollarDay("25.10.2026", "2.5600"))},
		{
			name: "date twice",
			body: answer(t, dollarDay("2026-10-25", "2.5600"), dollarDay("2026-10-25", "2.5500")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := dollarBuy(t, tt.body); !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("CommercialRates() error = %v, want kernel.ErrSourceUnavailable", err)
			}
		})
	}
}

// BASRATE-SC-007: A code with no price in one set, or with a price of zero, is missing.
func TestCommercialRatesReportAMissingCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buy, sell map[string]string
	}{
		{
			name: "buy zero",
			buy:  map[string]string{dollar: "2.5600", franc: "0.0000"},
			sell: map[string]string{dollar: dollarSell, franc: "3.2000"},
		},
		{
			name: "sell absent",
			buy:  map[string]string{dollar: "2.5600", franc: "3.1000"},
			sell: map[string]string{dollar: dollarSell},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := answer(t, day{date: "2026-10-25", buy: tt.buy, sell: tt.sell})

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

// BASRATE-SC-008: A broken price or a failure of the vendor wraps
// kernel.ErrSourceUnavailable.
func TestCommercialRatesReportABrokenSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "status 500", status: http.StatusInternalServerError, body: "down"},
		{
			name:   "buy not a number",
			status: http.StatusOK,
			body:   answer(t, dollarDay("2026-10-25", "n/a")),
		},
		{
			name:   "buy below zero",
			status: http.StatusOK,
			body:   answer(t, dollarDay("2026-10-25", "-2.5600")),
		},
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

// BASRATE-SC-010: The bank is Basisbank.
func TestBankIsBasisbank(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, http.StatusOK, recorded(t)).Bank()
	want := domain.Bank{Code: "basisbank", Name: "Basisbank", Kind: domain.KindBank}

	if got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
