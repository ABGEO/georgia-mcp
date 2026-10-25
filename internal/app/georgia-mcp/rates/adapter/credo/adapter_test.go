package credo_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/credo"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	credoclient "github.com/abgeo/georgia-mcp/pkg/credo"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata = "../../../../../../pkg/credo/testdata"
	testKey  = "secret-test-key"
	dollar   = "USD"
	franc    = "CHF"
	yen      = "JPY"

	dollarItem = `{"fromCcy":"USD","toCcy":"GEL","multiplier":1,"buy":2.593,"sell":2.607}`
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 8, 42, 36, 0, time.UTC)
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
	return `{"success":true,"rates":[` + strings.Join(list, ",") + `]}`
}

// syncBuffer is a buffer that the handler of a logger writes from many goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p) //nolint:wrapcheck // A bytes.Buffer never fails.
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func adapterWith(t *testing.T, status int, body string, sink *syncBuffer) *credo.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := credoclient.New(vendorhttp.WithBaseURL(server.URL), credoclient.WithAPIKey(testKey))
	logger := slog.New(slog.NewJSONHandler(sink, nil))

	return credo.NewAdapter(client, readAt, logger)
}

func adapterFor(t *testing.T, status int, body string) *credo.Adapter {
	t.Helper()

	return adapterWith(t, status, body, &syncBuffer{})
}

// CREDRATE-SC-005: The adapter gives the price of one unit, with the trailing zeros
// dropped.
func TestCommercialRatesGiveThePriceOfOneUnit(t *testing.T) {
	t.Parallel()

	result, err := adapterFor(t, http.StatusOK, recorded(t, "current-rates.json")).
		CommercialRates(t.Context(), []domain.CurrencyCode{dollar, "EUR"})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	got := map[domain.CurrencyCode][2]string{}
	for _, rate := range result.Rates {
		got[rate.Currency] = [2]string{rate.Buy.String(), rate.Sell.String()}
	}

	want := map[domain.CurrencyCode][2]string{
		"EUR":  {"2.875", "2.95"},
		dollar: {"2.593", "2.607"},
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

// CREDRATE-SC-006: The adapter divides a price of 100 units by 100.
func TestCommercialRatesDivideByTheMultiplier(t *testing.T) {
	t.Parallel()

	body := items(
		dollarItem,
		`{"fromCcy":"JPY","toCcy":"GEL","multiplier":100,"buy":1.7,"sell":2.1}`,
	)

	result, err := adapterFor(t, http.StatusOK, body).
		CommercialRates(t.Context(), []domain.CurrencyCode{yen})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	if len(result.Rates) != 1 {
		t.Fatalf("Rates = %v, want one rate", result.Rates)
	}

	rate := result.Rates[0]
	if rate.Buy.String() != "0.017" || rate.Sell.String() != "0.021" {
		t.Errorf("JPY = %s / %s, want 0.017 / 0.021", rate.Buy, rate.Sell)
	}
}

// CREDRATE-SC-007: A code with no item against GEL, or with a price that is absent or
// zero, is missing.
func TestCommercialRatesReportAMissingCode(t *testing.T) {
	t.Parallel()

	for _, item := range []string{
		`{"fromCcy":"CHF","toCcy":"GEL","multiplier":1,"buy":0,"sell":3.2}`,
		`{"fromCcy":"CHF","toCcy":"GEL","multiplier":1,"buy":3.1}`,
		`{"fromCcy":"CHF","toCcy":"USD","multiplier":1,"buy":0.8,"sell":0.9}`,
	} {
		t.Run(item, func(t *testing.T) {
			t.Parallel()

			result, err := adapterFor(t, http.StatusOK, items(dollarItem, item)).
				CommercialRates(t.Context(), []domain.CurrencyCode{franc, "GBP", dollar})
			if err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %v, want the rate of USD", result.Rates)
			}

			if diff := cmp.Diff([]domain.CurrencyCode{franc, "GBP"}, result.Missing); diff != "" {
				t.Errorf("Missing mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// CREDRATE-SC-008: A broken answer or a failure of the vendor wraps
// kernel.ErrSourceUnavailable.
func TestCommercialRatesReportABrokenSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "not succeeded", status: http.StatusOK, body: recorded(t, "not-succeeded.json")},
		{name: "status 500", status: http.StatusInternalServerError, body: "down"},
		{
			name:   "buy not a number",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"buy":2.593`, `"buy":"n/a"`, 1)),
		},
		{
			name:   "buy below zero",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"buy":2.593`, `"buy":-1`, 1)),
		},
		{
			name:   "multiplier of 3",
			status: http.StatusOK,
			body:   items(strings.Replace(dollarItem, `"multiplier":1`, `"multiplier":3`, 1)),
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

// CREDRATE-SC-009: A refused key logs one error that names the variable and holds no
// key, and the error is an outage.
func TestARefusedKeyIsLogged(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}

	_, err := adapterWith(t, http.StatusUnauthorized, "Unauthorized", sink).
		CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("CommercialRates() error = %v, want kernel.ErrSourceUnavailable", err)
	}

	log := sink.String()
	if strings.Count(log, `"msg":"Credo Bank refused the key"`) != 1 ||
		!strings.Contains(log, `"variable":"GEOMCP_RATES_CREDO_API_KEY"`) ||
		!strings.Contains(log, `"level":"ERROR"`) {
		t.Errorf("log = %s, want one error record that names the variable", log)
	}

	if strings.Contains(log, testKey) {
		t.Errorf("log = %s, want no key", log)
	}
}

// CREDRATE-SC-011: The bank is Credo Bank.
func TestBankIsCredo(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, http.StatusOK, recorded(t, "current-rates.json")).Bank()
	want := domain.Bank{Code: "credo", Name: "Credo Bank", Kind: domain.KindBank}

	if got != want {
		t.Errorf("Bank() = %+v, want %+v", got, want)
	}
}
