package rico_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/rico"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	ricoclient "github.com/abgeo/georgia-mcp/pkg/rico"
)

const (
	testdata  = "../../../../../../pkg/rico/testdata"
	dollar    = "USD"
	franc     = "CHF"
	yen       = "JPY"
	buyLabel  = "&#x10E7;&#x10D8;&#x10D3;&#x10D5;&#x10D0;"
	sellLabel = "&#x10D2;&#x10D0;&#x10E7;&#x10D8;&#x10D3;&#x10D5;&#x10D0;"
)

func readAt() time.Time {
	return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
}

func page(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdata, "home-ka.html")

	body, err := os.ReadFile(path) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// row returns a row of the table in the markup of the page.
func row(code, unit, buy, sell string) string {
	return `<div class="currency-row"><div class="currency-info"><strong>` + code +
		`</strong></div><div class="currency-unit"><span>u</span><strong>` + unit +
		`</strong></div><div class="currency-rate"><span>` + buyLabel + `</span><strong>` + buy +
		`</strong></div><div class="currency-rate"><span>` + sellLabel + `</span><strong>` + sell +
		`</strong></div></div>`
}

func table(rows ...string) string {
	return `<html><body><div class="currencies-table-body">` + strings.Join(rows, "") +
		`</div></body></html>`
}

func adapterFor(t *testing.T, status int, body string) *rico.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return rico.NewAdapter(ricoclient.New(ricoclient.WithBaseURL(server.URL)), readAt)
}

// RICORATE-SC-005: The adapter gives the price of one unit, with the trailing zeros dropped.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, page(t)).
		CommercialRates(t.Context(), []domain.CurrencyCode{"AMD", "PLN", dollar})
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
		"AMD":  {"0.0068", "0.0071"},
		"PLN":  {"0.61", "0.67"},
		dollar: {"2.596", "2.6"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}

	if len(result.Missing) != 0 || !result.ReadAt.Equal(readAt()) {
		t.Errorf("Missing = %v, ReadAt = %v, want none and %v",
			result.Missing, result.ReadAt, readAt())
	}
}

// RICORATE-SC-006: The cross rate is skipped. A code with no row, or a price of zero,
// is missing.
func TestCommercialRatesNameAMissingCode(t *testing.T) {
	t.Parallel()

	body := table(
		row(dollar, "1", "2.5960", "2.6000"),
		row("USD-EUR", "1", "1.1204", "1.1260"),
		row(franc, "1", "0.0000", "3.1510"),
	)

	result, err := adapterFor(t, http.StatusOK, body).
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
}

// RICORATE-SC-007: A unit that is not a power of 10, a price that is not a number or
// is below zero, and a failed call fail the bank.
func TestCommercialRatesFailOnABrokenRow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "unit 3", status: http.StatusOK, body: table(row(dollar, "3", "2.5960", "2.6000"))},
		{
			name:   "unit ten",
			status: http.StatusOK,
			body:   table(row(dollar, "ten", "2.5960", "2.6000")),
		},
		{
			name:   "not a number",
			status: http.StatusOK,
			body:   table(row(dollar, "1", "n/a", "2.6000")),
		},
		{
			name:   "below zero",
			status: http.StatusOK,
			body:   table(row(dollar, "1", "-2.5960", "2.6000")),
		},
		{name: "503", status: http.StatusServiceUnavailable},
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

	want := domain.Bank{Code: "rico", Name: "Rico Credit", Kind: domain.KindMicrofinance}
	if got := adapterFor(t, http.StatusOK, "").Bank(); got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
