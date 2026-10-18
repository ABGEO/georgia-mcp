package tbc_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tbc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	tbcclient "github.com/abgeo/georgia-mcp/pkg/tbc"
)

const (
	testdata = "../../../../../../pkg/tbc/testdata"
	dollar   = "USD"
)

func adapterFor(t *testing.T, handler http.HandlerFunc) *tbc.Adapter {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	readAt := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)

	return tbc.NewAdapter(
		tbcclient.New(tbcclient.WithBaseURL(server.URL), tbcclient.WithAPIKey("test-key")),
		func() time.Time { return readAt },
	)
}

func recorded(t *testing.T, status int, name string) http.HandlerFunc {
	t.Helper()

	path := filepath.Join(testdata, name)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

func answer(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }
}

func read(
	t *testing.T,
	handler http.HandlerFunc,
	codes ...domain.CurrencyCode,
) app.CommercialRatesResult {
	t.Helper()

	result, err := adapterFor(t, handler).CommercialRates(t.Context(), codes)
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	return result
}

func exact(value decimal.Decimal) string {
	return value.StringFixed(-value.Exponent())
}

// TBCRATE-SC-012: A rate keeps the digits of the answer, a trailing zero included.
func TestCommercialRatesKeepTheDigits(t *testing.T) {
	t.Parallel()

	result := read(t, answer(`{"base":"GEL","commercialRatesList":[`+
		`{"currency":"USD","buy":2.5800,"sell":2.6200},`+
		`{"currency":"AZN","buy":1.35,"sell":1.715}]}`), dollar, "AZN")

	got := make([]string, 0, 2*len(result.Rates))
	for _, rate := range result.Rates {
		got = append(got, exact(rate.Buy), exact(rate.Sell))
	}

	if diff := cmp.Diff([]string{"1.35", "1.715", "2.5800", "2.6200"}, got); diff != "" {
		t.Errorf("rates mismatch (-want +got):\n%s", diff)
	}
}

// TBCRATE-SC-013: A failure of TBC wraps kernel.ErrSourceUnavailable.
func TestCommercialRatesWrapAFailure(t *testing.T) {
	t.Parallel()

	tests := map[string]http.HandlerFunc{
		"401":                  recorded(t, http.StatusUnauthorized, "problem-401.json"),
		"base that is not GEL": answer(`{"base":"USD","commercialRatesList":[]}`),
		"rate that is not a number": answer(
			`{"base":"GEL","commercialRatesList":[{"currency":"USD","buy":"n/a","sell":2.6}]}`,
		),
	}

	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := adapterFor(
				t,
				handler,
			).CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("CommercialRates() error = %v, want ErrSourceUnavailable", err)
			}
		})
	}
}

// TBCRATE-SC-014: The rates of a recorded answer are prices of one unit, and a code
// that TBC does not quote is missing.
func TestCommercialRatesMapARecordedAnswer(t *testing.T) {
	t.Parallel()

	result := read(
		t,
		recorded(t, http.StatusOK, "commercial.json"),
		dollar,
		"EUR",
		"GBP",
		"JPY",
		"XXX",
	)

	codes := make([]domain.CurrencyCode, 0, len(result.Rates))
	for _, rate := range result.Rates {
		codes = append(codes, rate.Currency)

		if rate.Currency == "JPY" && exact(rate.Buy) != "0.0157" {
			t.Errorf("JPY buy = %s, want 0.0157", exact(rate.Buy))
		}
	}

	if diff := cmp.Diff([]domain.CurrencyCode{"EUR", "GBP", "JPY", dollar}, codes); diff != "" {
		t.Errorf("codes mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]domain.CurrencyCode{"XXX"}, result.Missing); diff != "" {
		t.Errorf("Missing mismatch (-want +got):\n%s", diff)
	}

	if result.ReadAt.IsZero() || result.Stale {
		t.Errorf(
			"ReadAt = %v, Stale = %v, want the read instant and false",
			result.ReadAt,
			result.Stale,
		)
	}
}

func TestCommercialRatesIgnoreACodeThatWasNotRequested(t *testing.T) {
	t.Parallel()

	result := read(t, answer(`{"base":"GEL","commercialRatesList":[`+
		`{"currency":"AVD","buy":1.709,"sell":1.899},{"currency":"USD","buy":2.566,"sell":2.628}]}`), dollar)

	if len(result.Rates) != 1 || result.Rates[0].Currency != "USD" {
		t.Errorf("Rates = %+v, want USD only", result.Rates)
	}
}

func TestBank(t *testing.T) {
	t.Parallel()

	want := domain.Bank{Code: "tbc", Name: "TBC Bank"}
	if got := adapterFor(t, answer("{}")).Bank(); got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}

// TBCRATE-SC-019: The error of a failed call holds no key.
func TestCommercialRatesKeepTheKeyOutOfTheError(t *testing.T) {
	t.Parallel()

	const key = "secret-tbc-key"

	server := httptest.NewServer(recorded(t, http.StatusUnauthorized, "problem-401.json"))
	t.Cleanup(server.Close)

	adapter := tbc.NewAdapter(
		tbcclient.New(tbcclient.WithBaseURL(server.URL), tbcclient.WithAPIKey(key)),
		time.Now,
	)

	_, err := adapter.CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
	if err == nil || strings.Contains(fmt.Sprintf("%+v", err), key) {
		t.Errorf("CommercialRates() error = %v, want an error with no key", err)
	}
}
