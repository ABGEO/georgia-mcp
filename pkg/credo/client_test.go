package credo_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/credo"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const dollar = "USD"

func recorded(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

func serve(t *testing.T, status int, body []byte) *credo.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return credo.New(vendorhttp.WithBaseURL(server.URL), credo.WithAPIKey("k1"))
}

func currentRates(t *testing.T, client *credo.Client) ([]credo.Rate, error) {
	t.Helper()

	return client.CurrentRates(t.Context(), credo.MethodCommercial, time.Now())
}

// CREDRATE-SC-001: The client sends a GET with the method, the instant in UTC with
// milliseconds, the key in x-api-key, and the User-Agent of the repository.
func TestCurrentRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write(recorded(t, "current-rates.json"))
	}))
	t.Cleanup(server.Close)

	client := credo.New(vendorhttp.WithBaseURL(server.URL), credo.WithAPIKey("k1"))
	at := time.Date(2026, 10, 25, 12, 42, 36, 0, time.FixedZone("Tbilisi", 4*60*60))

	if _, err := client.CurrentRates(t.Context(), credo.MethodCommercial, at); err != nil {
		t.Fatalf("CurrentRates() error = %v", err)
	}

	for name, pair := range map[string][2]string{
		"method":    {got.Method, http.MethodGet},
		"path":      {got.URL.Path, "/v2/credo/current-rates"},
		"?method":   {got.URL.Query().Get("method"), "FxRatesCommercial"},
		"?date":     {got.URL.Query().Get("date"), "2026-10-25T08:42:36.000Z"},
		"x-api-key": {got.Header.Get("X-Api-Key"), "k1"},
		"Accept":    {got.Header.Get("Accept"), "application/json"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}
}

// CREDRATE-SC-001: An empty key sends no header x-api-key.
func TestWithAPIKeyOfNoKeySendsNoHeader(t *testing.T) {
	t.Parallel()

	var header []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Values("X-Api-Key")
		_, _ = w.Write(recorded(t, "current-rates.json"))
	}))
	t.Cleanup(server.Close)

	client := credo.New(vendorhttp.WithBaseURL(server.URL), credo.WithAPIKey(""))
	if _, err := currentRates(t, client); err != nil {
		t.Fatalf("CurrentRates() error = %v", err)
	}

	if header != nil {
		t.Errorf("x-api-key = %q, want no header", header)
	}
}

// CREDRATE-SC-002: The client reads 3 items from the recorded answer.
func TestCurrentRatesReadsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	rates, err := currentRates(t, serve(t, http.StatusOK, recorded(t, "current-rates.json")))
	if err != nil {
		t.Fatalf("CurrentRates() error = %v", err)
	}

	if len(rates) != 3 {
		t.Fatalf("CurrentRates() = %d items, want 3", len(rates))
	}

	want := credo.Rate{FromCcy: dollar, ToCcy: "GEL", Multiplier: "1", Buy: "2.593", Sell: "2.607"}
	for _, rate := range rates {
		if rate.FromCcy == dollar && rate != want {
			t.Errorf("item of USD = %+v, want %+v", rate, want)
		}
	}
}

// CREDRATE-SC-003: A false success is ErrNotSucceeded.
func TestCurrentRatesReportsNotSucceeded(t *testing.T) {
	t.Parallel()

	rates, err := currentRates(t, serve(t, http.StatusOK, recorded(t, "not-succeeded.json")))
	if rates != nil || !errors.Is(err, credo.ErrNotSucceeded) {
		t.Errorf("CurrentRates() = %v, %v, want nil and ErrNotSucceeded", rates, err)
	}
}

// CREDRATE-SC-003: A 401 and a 500 are an APIError with the first 1 KiB of the body.
func TestCurrentRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "401", status: http.StatusUnauthorized, body: "Unauthorized", want: "Unauthorized"},
		{
			name:   "500",
			status: http.StatusInternalServerError,
			body:   strings.Repeat("x", 2048),
			want:   strings.Repeat("x", 1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := currentRates(t, serve(t, tt.status, []byte(tt.body)))

			apiErr, ok := errors.AsType[*credo.APIError](err)
			if !ok {
				t.Fatalf("CurrentRates() error = %v, want an APIError", err)
			}

			if apiErr.StatusCode != tt.status || apiErr.Body != tt.want {
				t.Errorf(
					"APIError = %d with %d bytes, want %d with %d",
					apiErr.StatusCode,
					len(apiErr.Body),
					tt.status,
					len(tt.want),
				)
			}

			if !errors.Is(err, credo.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
			}
		})
	}
}
