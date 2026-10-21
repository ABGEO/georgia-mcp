package terabank_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/terabank"
)

const dollar = "USD"

func serve(t *testing.T, status int, body string) *terabank.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return terabank.New(terabank.WithBaseURL(server.URL))
}

func recorded(t *testing.T, name string) string {
	t.Helper()

	body, err := os.ReadFile("testdata/" + name) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// TERARATE-SC-001: The client sends a POST with an empty body of length 0, the Accept
// header, the User-Agent, and no key.
func TestCrossRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var (
		got    *http.Request
		length []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		length = r.Header.Values("Content-Length")

		_, _ = w.Write([]byte(`{"data":[],"succeeded":true,"message":""}`))
	}))
	t.Cleanup(server.Close)

	if _, err := terabank.New(terabank.WithBaseURL(server.URL)).
		CrossRates(t.Context()); err != nil {
		t.Fatalf("CrossRates() error = %v", err)
	}

	if got.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", got.Method)
	}

	if want := "/_mvcapi/CurrencyRatesApi/GetTeraCrossRates"; got.URL.Path != want {
		t.Errorf("path = %q, want %q", got.URL.Path, want)
	}

	if got.ContentLength != 0 || len(length) != 1 || length[0] != "0" {
		t.Errorf("Content-Length = %v (%d), want 0", length, got.ContentLength)
	}

	assertHeaders(t, got.Header)
}

func assertHeaders(t *testing.T, header http.Header) {
	t.Helper()

	if accept := header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-terabank/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-terabank/<version> (+<repository>)", agent)
	}

	for _, name := range []string{"Apikey", "Authorization"} {
		if value := header.Get(name); value != "" {
			t.Errorf("%s = %q, want no key header", name, value)
		}
	}
}

// TERARATE-SC-002: The client decodes a recorded answer and keeps its digits.
func TestCrossRatesDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	rates, err := serve(t, http.StatusOK, recorded(t, "cross-rates.json")).CrossRates(t.Context())
	if err != nil {
		t.Fatalf("CrossRates() error = %v", err)
	}

	if len(rates) != 3 {
		t.Fatalf("CrossRates() = %d items, want 3", len(rates))
	}

	want := terabank.CrossRate{
		ISO:                  dollar,
		Buy:                  "2.5960",
		Sell:                 "2.6010",
		PreviousDayBuy:       "2.5470",
		PreviousDaySell:      "2.6470",
		ShowPreviousDayRates: true,
	}
	if rates[0] != want {
		t.Errorf("CrossRates()[0] = %+v, want %+v", rates[0], want)
	}
}

// TERARATE-SC-003: An answer that did not succeed is ErrNotSucceeded with its message.
func TestCrossRatesReportsAFailedAnswer(t *testing.T) {
	t.Parallel()

	rates, err := serve(t, http.StatusOK, recorded(t, "not-succeeded.json")).CrossRates(t.Context())
	if rates != nil || !errors.Is(err, terabank.ErrNotSucceeded) {
		t.Fatalf("CrossRates() = %v, %v, want nil and ErrNotSucceeded", rates, err)
	}

	if !strings.Contains(err.Error(), "error") {
		t.Errorf("error = %q, want the message of the answer", err)
	}
}

// TERARATE-SC-003: A 411 is an APIError with the first 1 KiB of the body.
func TestCrossRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusLengthRequired, strings.Repeat("x", 2048)).CrossRates(t.Context())

	apiErr, ok := errors.AsType[*terabank.APIError](err)
	if !ok {
		t.Fatalf("CrossRates() error = %v, want an APIError", err)
	}

	want := terabank.APIError{
		StatusCode: http.StatusLengthRequired,
		Body:       strings.Repeat("x", 1024),
	}
	if *apiErr != want {
		t.Errorf("APIError = %+v, want %+v", *apiErr, want)
	}

	if !errors.Is(err, terabank.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}

func TestAPIErrorText(t *testing.T) {
	t.Parallel()

	err := &terabank.APIError{StatusCode: http.StatusLengthRequired}
	if got, want := err.Error(), "terabank: unexpected status: 411: Length Required"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
