package socar_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/socar"
)

func serveFile(t *testing.T, path string, status int) *httptest.Server {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server
}

// SOCFUEL-SC-001: The client sends the path, the Accept header, and the User-Agent.
func TestCurrentPricesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	if _, err := socar.New(socar.WithBaseURL(server.URL)).CurrentPrices(t.Context()); err != nil {
		t.Fatalf("CurrentPrices() error = %v", err)
	}

	if got.URL.Path != "/integration/info/current-prices" {
		t.Errorf("path = %q, want /integration/info/current-prices", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-socar/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-socar/<version> (+<repository>)", agent)
	}
}

// SOCFUEL-SC-001: The client decodes a recorded answer and keeps the digits and the
// names of the answer.
func TestCurrentPricesDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/current-prices.json", http.StatusOK)

	got, err := socar.New(socar.WithBaseURL(server.URL)).CurrentPrices(t.Context())
	if err != nil {
		t.Fatalf("CurrentPrices() error = %v", err)
	}

	if got.Status != "1" || len(got.Results) != 7 {
		t.Fatalf("Status = %q with %d products, want 1 with 7", got.Status, len(got.Results))
	}

	byCode := make(map[string]socar.FuelPrice, len(got.Results))
	for _, price := range got.Results {
		byCode[price.FuelCode] = price
	}

	tests := map[string]struct{ english, price string }{
		"EURREG": {english: "Nano Euro Regular", price: "4.02"},
		"LPG":    {english: "LPG ", price: "2.09"},
		"DIESEL": {english: "Euro 5 Diesel", price: "4.9"},
	}

	for code, want := range tests {
		price := byCode[code]
		if price.FuelNameEng != want.english || price.FuelUnitPrice.String() != want.price {
			t.Errorf(
				"%s = %q at %s, want %q at %s",
				code, price.FuelNameEng, price.FuelUnitPrice, want.english, want.price,
			)
		}
	}
}

// SOCFUEL-SC-002: A status other than 200 is an APIError with the first 1 KiB of the body.
func TestCurrentPricesReportsAStatus(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/error.json")
	if err != nil {
		t.Fatal(err)
	}

	server := serveFile(t, "testdata/error.json", http.StatusBadRequest)

	_, err = socar.New(socar.WithBaseURL(server.URL)).CurrentPrices(t.Context())

	apiErr, ok := errors.AsType[*socar.APIError](err)
	if !ok {
		t.Fatalf("CurrentPrices() error = %v, want *APIError", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}

	if want := string(body[:min(len(body), 1024)]); apiErr.Body != want {
		t.Errorf("Body has %d bytes, want %d", len(apiErr.Body), len(want))
	}

	if !errors.Is(err, socar.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
