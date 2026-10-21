package bog_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/bog"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	bogclient "github.com/abgeo/georgia-mcp/pkg/bog"
)

const (
	testdata = "../../../../../../pkg/bog/testdata"
	dollar   = "USD"
	unknown  = "XXX"
)

func readAt() time.Time {
	return time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
}

func adapterFor(t *testing.T, handler http.Handler) *bog.Adapter {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return bog.NewAdapter(
		bogclient.New(bogclient.WithBaseURL(server.URL)),
		readAt,
	)
}

func recorded(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// answers serves the body of each code, and 204 for a code it does not hold.
func answers(bodies map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[path.Base(r.URL.Path)]
		if !ok {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		_, _ = w.Write([]byte(body))
	}
}

func read(
	t *testing.T,
	handler http.Handler,
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

// BOGRATE-SC-005: A rate drops the padding zeros of the answer and keeps every other digit.
func TestCommercialRatesDropThePadding(t *testing.T) {
	t.Parallel()

	tests := map[domain.CurrencyCode]struct {
		file      string
		buy, sell string
	}{
		dollar: {file: "commercial-usd.json", buy: "2.565", sell: "2.629"},
		"JPY":  {file: "commercial-jpy.json", buy: "0.01577", sell: "0.01713"},
		"IDR":  {file: "commercial-idr.json", buy: "0.000126", sell: "0.000163"},
	}

	for code, tt := range tests {
		t.Run(string(code), func(t *testing.T) {
			t.Parallel()

			result := read(
				t,
				answers(map[string]string{string(code): string(recorded(t, tt.file))}),
				code,
			)
			if len(result.Rates) != 1 {
				t.Fatalf("Rates = %+v, want one rate", result.Rates)
			}

			got := []string{exact(result.Rates[0].Buy), exact(result.Rates[0].Sell)}
			if diff := cmp.Diff([]string{tt.buy, tt.sell}, got); diff != "" {
				t.Errorf("buy and sell mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// BOGRATE-SC-006: A code with no rate, or with a price of zero, is missing.
func TestCommercialRatesListACodeWithNoRateAsMissing(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"204":           "",
		"zero buy":      `{"Buy":0,"Sell":2.629}`,
		"no sell":       `{"Buy":2.565}`,
		"zero sell":     `{"Buy":2.565,"Sell":0.00000000}`,
		"empty answers": `{}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bodies := map[string]string{dollar: string(recorded(t, "commercial-usd.json"))}
			if body != "" {
				bodies[unknown] = body
			}

			result := read(t, answers(bodies), dollar, unknown)

			if len(result.Rates) != 1 || result.Rates[0].Currency != dollar {
				t.Errorf("Rates = %+v, want USD only", result.Rates)
			}

			if diff := cmp.Diff([]domain.CurrencyCode{unknown}, result.Missing); diff != "" {
				t.Errorf("Missing mismatch (-want +got):\n%s", diff)
			}

			if !result.ReadAt.Equal(readAt()) || result.Stale {
				t.Errorf(
					"ReadAt = %v, Stale = %v, want %v and false",
					result.ReadAt,
					result.Stale,
					readAt(),
				)
			}
		})
	}
}

// BOGRATE-SC-007: A failure of Bank of Georgia wraps kernel.ErrSourceUnavailable, after
// one call for the code.
func TestCommercialRatesWrapAFailure(t *testing.T) {
	t.Parallel()

	tests := map[string]http.HandlerFunc{
		"429": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusTooManyRequests)
		},
		"500": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
		"price that is not a number": answers(
			map[string]string{dollar: `{"Buy":"n/a","Sell":2.629}`},
		),
		"price below zero": answers(map[string]string{dollar: `{"Buy":-1,"Sell":2.629}`}),
	}

	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			counting := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				handler(w, r)
			})

			_, err := adapterFor(
				t,
				counting,
			).CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("CommercialRates() error = %v, want ErrSourceUnavailable", err)
			}

			if got := calls.Load(); got != 1 {
				t.Errorf("calls = %d, want 1", got)
			}
		})
	}
}

// BOGRATE-SC-008: The adapter reads the codes at once. A vendor that answers only when
// 10 calls wait would block a reader of one call at a time.
func TestCommercialRatesReadTheCodesAtOnce(t *testing.T) {
	t.Parallel()

	codes := []domain.CurrencyCode{
		"AUD", "CAD", "CHF", "CNY", "EUR", "GBP", "JPY", "SEK", "TRY", dollar,
	}

	var arrived atomic.Int32

	everyCall := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if int(arrived.Add(1)) == len(codes) {
			close(everyCall)
		}

		select {
		case <-everyCall:
			_, _ = w.Write([]byte(`{"Buy":2.56500000,"Sell":2.62900000}`))
		case <-r.Context().Done():
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	adapter := bog.NewAdapter(
		bogclient.New(
			bogclient.WithBaseURL(server.URL),
			bogclient.WithHTTPClient(&http.Client{Timeout: 5 * time.Second}),
		),
		readAt,
	)

	result, err := adapter.CommercialRates(t.Context(), codes)
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	got := make([]domain.CurrencyCode, 0, len(result.Rates))
	for _, rate := range result.Rates {
		got = append(got, rate.Currency)
	}

	if diff := cmp.Diff(codes, got); diff != "" {
		t.Errorf("codes mismatch (-want +got):\n%s", diff)
	}

	want := domain.Bank{Code: "bog", Name: "Bank of Georgia", Kind: domain.KindBank}
	if bank := adapter.Bank(); bank != want {
		t.Errorf("Bank() = %+v, want %+v", bank, want)
	}
}

func TestCommercialRatesNameTheCodeInAnError(t *testing.T) {
	t.Parallel()

	_, err := adapterFor(t, answers(map[string]string{dollar: `{"Buy":"n/a","Sell":1}`})).
		CommercialRates(t.Context(), []domain.CurrencyCode{dollar})
	if err == nil || !strings.Contains(err.Error(), dollar) {
		t.Errorf("CommercialRates() error = %v, want an error that names USD", err)
	}
}
