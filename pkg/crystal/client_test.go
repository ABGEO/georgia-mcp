package crystal_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/crystal"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	dollar   = "USD"
	lari     = "GEL"
	cashRate = "web-rate-cash.json"
)

func recorded(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", name)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// edited returns the recorded answer with one text replaced, and fails when the text is
// not in it.
func edited(t *testing.T, old, replacement string) string {
	t.Helper()

	body := recorded(t, cashRate)
	if !strings.Contains(body, old) {
		t.Fatalf("the recorded answer holds no %s", old)
	}

	return strings.Replace(body, old, replacement, 1)
}

func serve(t *testing.T, status int, body string) *crystal.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return crystal.New(vendorhttp.WithBaseURL(server.URL))
}

// CRYSRATE-SC-001: The client asks for the cash rates as JSON, with the User-Agent of
// the repository, no product, no city, and no token.
func TestCashRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write([]byte(recorded(t, cashRate)))
	}))
	t.Cleanup(server.Close)

	if _, err := crystal.New(vendorhttp.WithBaseURL(server.URL)).
		CashRates(t.Context()); err != nil {
		t.Fatalf("CashRates() error = %v", err)
	}

	for name, pair := range map[string][2]string{
		"method": {got.Method, http.MethodGet},
		"path":   {got.URL.Path, "/api/Conversion/GetWebRateByProductID"},
		"query":  {got.URL.RawQuery, "IsCash=true"},
		"Accept": {got.Header.Get("Accept"), "application/json"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}

	if agent := got.Header.Get("User-Agent"); !strings.HasPrefix(agent, "georgia-mcp/") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version>", agent)
	}

	if token := got.Header.Get("Authorization"); token != "" {
		t.Errorf("Authorization = %q, want no header", token)
	}
}

// CRYSRATE-SC-002: The client reads the 11 items of the recorded answer, with the exact
// digits of each price.
func TestCashRatesReadsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	rates, err := serve(t, http.StatusOK, recorded(t, cashRate)).CashRates(t.Context())
	if err != nil {
		t.Fatalf("CashRates() error = %v", err)
	}

	if len(rates) != 11 {
		t.Fatalf("CashRates() = %d items, want 11", len(rates))
	}

	for _, want := range []crystal.Rate{
		{SourceISO: dollar, TargetISO: lari, Multiplier: "1", BuyRate: "2.595", SellRate: "2.599"},
		{SourceISO: "ILS", TargetISO: lari, Multiplier: "10", BuyRate: "3.5", SellRate: "8"},
	} {
		found := false

		for _, rate := range rates {
			if rate == want {
				found = true
			}
		}

		if !found {
			t.Errorf("CashRates() = %+v, want an item %+v", rates, want)
		}
	}
}

// CRYSRATE-SC-003: A failure in either envelope is ErrNotSucceeded. An empty list is not
// an error.
func TestCashRatesReadBothEnvelopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "inner success false", body: recorded(t, "not-succeeded.json")},
		{
			name: "outer error code",
			body: edited(
				t,
				`{"errorCode":"0","errorText":""`,
				`{"errorCode":"500","errorText":"down"`,
			),
		},
		{
			name: "inner error code",
			body: edited(t, `"data":{"errorCode":"0"`, `"data":{"errorCode":"1"`),
		},
		{name: "no inner envelope", body: `{"errorCode":"0","errorText":""}`},
		{
			name: "null list",
			body: `{"errorCode":"0","errorText":"","data":{"errorCode":"0","success":true,"data":null}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rates, err := serve(t, http.StatusOK, tt.body).CashRates(t.Context())
			if rates != nil || !errors.Is(err, crystal.ErrNotSucceeded) {
				t.Errorf("CashRates() = %v, %v, want nil and ErrNotSucceeded", rates, err)
			}
		})
	}

	empty := edited(t, `"data":[`, `"data":[],"was":[`)

	rates, err := serve(t, http.StatusOK, empty).CashRates(t.Context())
	if err != nil || len(rates) != 0 {
		t.Errorf("CashRates() = %v, %v, want an empty list and no error", rates, err)
	}
}

// CRYSRATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestCashRatesReportAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(
		t,
		http.StatusServiceUnavailable,
		strings.Repeat("x", 2048),
	).CashRates(t.Context())

	apiErr, ok := errors.AsType[*crystal.APIError](err)
	if !ok {
		t.Fatalf("CashRates() error = %v, want an APIError", err)
	}

	if apiErr.StatusCode != http.StatusServiceUnavailable ||
		apiErr.Body != strings.Repeat("x", 1024) {
		t.Errorf(
			"APIError = %d with %d bytes, want 503 with 1024",
			apiErr.StatusCode,
			len(apiErr.Body),
		)
	}

	if !errors.Is(err, crystal.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
