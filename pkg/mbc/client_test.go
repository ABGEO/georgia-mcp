package mbc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/mbc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	dollar = "USD"
	lari   = "GEL"
)

func recorded(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

func serve(t *testing.T, status int, body []byte) *mbc.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return mbc.New(vendorhttp.WithBaseURL(server.URL))
}

// MBCRATE-SC-001: The client sends a GET with no query, as JSON, with the User-Agent of
// the repository and no key.
func TestCommercialRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write(recorded(t, "commercial.json"))
	}))
	t.Cleanup(server.Close)

	if _, err := mbc.New(vendorhttp.WithBaseURL(server.URL)).
		CommercialRates(t.Context()); err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	for name, pair := range map[string][2]string{
		"method": {got.Method, http.MethodGet},
		"path":   {got.URL.Path, "/api/fxrates/mbc/commercial"},
		"query":  {got.URL.RawQuery, ""},
		"Accept": {got.Header.Get("Accept"), "application/json"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version>", agent)
	}

	for _, name := range []string{"Apikey", "Authorization", "X-Api-Key"} {
		if value := got.Header.Get(name); value != "" {
			t.Errorf("%s = %q, want no key header", name, value)
		}
	}
}

// MBCRATE-SC-002: The client reads the 5 items of the recorded answer, with the exact
// digits of each price.
func TestCommercialRatesReadsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	rates, err := serve(
		t,
		http.StatusOK,
		recorded(t, "commercial.json"),
	).CommercialRates(t.Context())
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	if len(rates) != 5 {
		t.Fatalf("CommercialRates() = %d items, want 5", len(rates))
	}

	want := mbc.Rate{FromCcy: dollar, ToCcy: lari, Multiplier: "1", Buy: "2.5950", Sell: "2.5990"}

	for _, rate := range rates {
		if rate.FromCcy == dollar && rate != want {
			t.Errorf("item of USD = %+v, want %+v", rate, want)
		}
	}
}

// MBCRATE-SC-003: An answer with no list is an error. An empty list is not.
func TestCommercialRatesReadTheList(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusOK, recorded(t, "no-rates.json")).CommercialRates(t.Context())
	if !errors.Is(err, mbc.ErrNoRates) {
		t.Errorf("CommercialRates() error = %v, want ErrNoRates", err)
	}

	rates, err := serve(t, http.StatusOK, []byte(`{"FXRates":[],"timestamp":1}`)).
		CommercialRates(t.Context())
	if err != nil || len(rates) != 0 {
		t.Errorf("CommercialRates() = %v, %v, want an empty list and no error", rates, err)
	}
}

// MBCRATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestCommercialRatesReportAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusServiceUnavailable, []byte(strings.Repeat("x", 2048))).
		CommercialRates(t.Context())

	apiErr, ok := errors.AsType[*mbc.APIError](err)
	if !ok {
		t.Fatalf("CommercialRates() error = %v, want an APIError", err)
	}

	if apiErr.StatusCode != http.StatusServiceUnavailable ||
		apiErr.Body != strings.Repeat("x", 1024) {
		t.Errorf(
			"APIError = %d with %d bytes, want 503 with 1024",
			apiErr.StatusCode,
			len(apiErr.Body),
		)
	}

	if !errors.Is(err, mbc.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
