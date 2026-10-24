package neogas_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/neogas"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func serve(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// NEOFUEL-SC-001: The client sends the path, the Accept header, and the User-Agent, and
// returns the text of each gas price of the page.
func TestGasPricesReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/home.html")
	if err != nil {
		t.Fatal(err)
	}

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	prices, err := neogas.New(vendorhttp.WithBaseURL(server.URL)).GasPrices(t.Context())
	if err != nil {
		t.Fatalf("GasPrices() error = %v", err)
	}

	if want := []string{"1.60", "1.60"}; !slices.Equal(prices, want) {
		t.Errorf("GasPrices() = %q, want %q", prices, want)
	}

	if got.URL.Path != "/ka/" {
		t.Errorf("path = %q, want /ka/", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}
}

// NEOFUEL-SC-002: A page with no gas price gives an empty list, and a status other than
// 200 is an APIError.
func TestGasPricesMirrorsThePage(t *testing.T) {
	t.Parallel()

	prices, err := neogas.New(vendorhttp.WithBaseURL(serve(t, "<html><body></body></html>", 200).URL)).
		GasPrices(t.Context())
	if err != nil || len(prices) != 0 {
		t.Errorf("GasPrices() = %q, %v, want an empty list and no error", prices, err)
	}

	_, err = neogas.New(vendorhttp.WithBaseURL(serve(t, "busy", http.StatusServiceUnavailable).URL)).
		GasPrices(t.Context())

	apiErr, ok := errors.AsType[*neogas.APIError](err)
	if !ok || apiErr.StatusCode != http.StatusServiceUnavailable || apiErr.Body != "busy" {
		t.Fatalf("GasPrices() error = %v, want an *APIError with 503 and the body", err)
	}

	if !errors.Is(err, neogas.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
