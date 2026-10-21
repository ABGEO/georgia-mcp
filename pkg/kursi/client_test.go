package kursi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/kursi"
)

const dollar = "USD"

func serve(t *testing.T, status int, body string) *kursi.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return kursi.New(kursi.WithBaseURL(server.URL))
}

func recorded(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile("testdata/currencies.json")
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// KURSRATE-SC-001: The client sends a GET of the public method, with the Accept header,
// the User-Agent, and no token.
func TestCurrenciesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	if _, err := kursi.New(kursi.WithBaseURL(server.URL)).Currencies(t.Context()); err != nil {
		t.Fatalf("Currencies() error = %v", err)
	}

	if got.Method != http.MethodGet || got.URL.Path != "/api/public/currencies" {
		t.Errorf("request = %s %s, want GET /api/public/currencies", got.Method, got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-kursi/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-kursi/<version> (+<repository>)", agent)
	}

	if token := got.Header.Get("Authorization"); token != "" {
		t.Errorf("Authorization = %q, want none", token)
	}
}

// KURSRATE-SC-002: The client decodes a recorded answer and keeps its digits.
func TestCurrenciesDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	pairs, err := serve(t, http.StatusOK, recorded(t)).Currencies(t.Context())
	if err != nil {
		t.Fatalf("Currencies() error = %v", err)
	}

	if len(pairs) != 8 {
		t.Fatalf("Currencies() = %d pairs, want 8", len(pairs))
	}

	got := pairs[0]
	if got.BaseCurrencyCode != "GEL" || got.SecondaryCurrencyCode != dollar ||
		got.BuyRate != "2.5940" || got.SellRate != "2.6010" ||
		got.BankRates["BOG"].BuyRate != "2.5650" {
		t.Errorf("Currencies()[0] = %+v, want GEL and USD, 2.5940, 2.6010, BOG buy 2.5650", got)
	}
}

// KURSRATE-SC-003: A 500 is an APIError with the first 1 KiB of the body. A body that
// is not JSON is another error.
func TestCurrenciesReportsAFailure(t *testing.T) {
	t.Parallel()

	_, err := serve(
		t,
		http.StatusInternalServerError,
		strings.Repeat("x", 2048),
	).Currencies(t.Context())

	apiErr, ok := errors.AsType[*kursi.APIError](err)
	if !ok {
		t.Fatalf("Currencies() error = %v, want an APIError", err)
	}

	want := kursi.APIError{
		StatusCode: http.StatusInternalServerError,
		Body:       strings.Repeat("x", 1024),
	}
	if *apiErr != want || !errors.Is(err, kursi.ErrUnexpectedStatus) {
		t.Errorf("APIError = %+v, want %+v that wraps ErrUnexpectedStatus", *apiErr, want)
	}

	pairs, err := serve(t, http.StatusOK, "[").Currencies(t.Context())
	if _, isAPI := errors.AsType[*kursi.APIError](err); err == nil || isAPI || pairs != nil {
		t.Errorf("Currencies() = %v, %v, want nil and an error that is not an APIError", pairs, err)
	}
}

func TestAPIErrorText(t *testing.T) {
	t.Parallel()

	err := &kursi.APIError{StatusCode: http.StatusBadGateway}
	if got, want := err.Error(), "kursi: unexpected status: 502: Bad Gateway"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
