package bog_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/bog"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const dollar = "USD"

func serveFile(t *testing.T, path string) *httptest.Server {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server
}

// BOGRATE-SC-001: The client sends the path with the code, the Accept header, the
// User-Agent, and no key.
func TestCommercialRateSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`{"Buy":2.565,"Sell":2.629}`))
	}))
	t.Cleanup(server.Close)

	if _, err := bog.New(vendorhttp.WithBaseURL(server.URL)).
		CommercialRate(t.Context(), dollar); err != nil {
		t.Fatalf("CommercialRate() error = %v", err)
	}

	if got.URL.Path != "/api/rates/commercial/USD" {
		t.Errorf("path = %q, want %q", got.URL.Path, "/api/rates/commercial/USD")
	}

	if accept := got.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-bog/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-bog/<version> (+<repository>)", agent)
	}

	for _, name := range []string{"Apikey", "Authorization"} {
		if value := got.Header.Get(name); value != "" {
			t.Errorf("%s = %q, want no key header", name, value)
		}
	}
}

// BOGRATE-SC-002: The client decodes a recorded answer and keeps its digits.
func TestCommercialRateDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/commercial-usd.json")

	got, err := bog.New(vendorhttp.WithBaseURL(server.URL)).CommercialRate(t.Context(), dollar)
	if err != nil {
		t.Fatalf("CommercialRate() error = %v", err)
	}

	want := bog.CommercialRate{Buy: "2.56500000", Sell: "2.62900000"}
	if *got != want {
		t.Errorf("CommercialRate() = %+v, want %+v", *got, want)
	}
}

// BOGRATE-SC-003: A 204 is ErrNoRate.
func TestCommercialRateReportsNoRate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	rate, err := bog.New(vendorhttp.WithBaseURL(server.URL)).CommercialRate(t.Context(), "GEL")
	if rate != nil || !errors.Is(err, bog.ErrNoRate) {
		t.Errorf("CommercialRate() = %+v, %v, want nil and ErrNoRate", rate, err)
	}
}

// BOGRATE-SC-003: Another status is an APIError with the Retry-After header and the
// first 1 KiB of the body.
func TestCommercialRateReportsAStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		retryAfter string
		body       string
		want       *bog.APIError
	}{
		{
			name:       "429",
			status:     http.StatusTooManyRequests,
			retryAfter: "30",
			want:       &bog.APIError{StatusCode: http.StatusTooManyRequests, RetryAfter: "30"},
		},
		{
			name:   "500",
			status: http.StatusInternalServerError,
			body:   strings.Repeat("x", 2048),
			want: &bog.APIError{
				StatusCode: http.StatusInternalServerError,
				Body:       strings.Repeat("x", 1024),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if tt.retryAfter != "" {
						w.Header().Set("Retry-After", tt.retryAfter)
					}

					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
				}),
			)
			t.Cleanup(server.Close)

			_, err := bog.New(vendorhttp.WithBaseURL(server.URL)).
				CommercialRate(t.Context(), dollar)

			apiErr, ok := errors.AsType[*bog.APIError](err)
			if !ok {
				t.Fatalf("CommercialRate() error = %v, want an APIError", err)
			}

			if *apiErr != *tt.want {
				t.Errorf("APIError = %+v, want %+v", apiErr, tt.want)
			}

			if !errors.Is(err, bog.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
			}
		})
	}
}

func TestAPIErrorText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  *bog.APIError
		want string
	}{
		"with Retry-After": {
			err:  &bog.APIError{StatusCode: http.StatusTooManyRequests, RetryAfter: "30"},
			want: "bog: unexpected status: 429: Too Many Requests (retry after 30)",
		},
		"without Retry-After": {
			err:  &bog.APIError{StatusCode: http.StatusBadGateway, Body: "<html>"},
			want: "bog: unexpected status: 502: Bad Gateway",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommercialRateEscapesTheCode(t *testing.T) {
	t.Parallel()

	var got string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	_, _ = bog.New(vendorhttp.WithBaseURL(server.URL)).CommercialRate(t.Context(), "a/b?c")

	if got != "/api/rates/commercial/a%2Fb%3Fc" {
		t.Errorf("path = %q, want %q", got, "/api/rates/commercial/a%2Fb%3Fc")
	}
}
