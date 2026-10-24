package wissol_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
	"github.com/abgeo/georgia-mcp/pkg/wissol"
)

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

// WISFUEL-SC-001: The client sends the path, the Accept header, and the User-Agent.
func TestFuelPricesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	if _, err := wissol.New(vendorhttp.WithBaseURL(server.URL)).
		FuelPrices(t.Context()); err != nil {
		t.Fatalf("FuelPrices() error = %v", err)
	}

	if got.URL.Path != "/FuelPrice" {
		t.Errorf("path = %q, want /FuelPrice", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}
}

// WISFUEL-SC-001: The client decodes a recorded answer and keeps the digits of each price.
func TestFuelPricesDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/fuelprice.json")

	got, err := wissol.New(vendorhttp.WithBaseURL(server.URL)).FuelPrices(t.Context())
	if err != nil {
		t.Fatalf("FuelPrices() error = %v", err)
	}

	if len(got) != 8 {
		t.Fatalf("FuelPrices() holds %d products, want 8", len(got))
	}

	byName := make(map[string]wissol.FuelPrice, len(got))
	for _, price := range got {
		byName[price.FuelTypeEng] = price
	}

	type digest struct {
		code, georgian, price, selfService, date string
		active                                   bool
	}

	tests := map[string]digest{
		"Euro Regular": {
			code: "05", georgian: "ევრო რეგულარი", price: "4.08", selfService: "3.93",
			date: "2026-10-08T00:00:00", active: true,
		},
		"A1 SUPER 100": {
			code: "07", georgian: "A1 სუპერი 100", price: "4.79",
			date: "2026-10-17T00:00:00", active: true,
		},
	}

	for name, want := range tests {
		price := byName[name]

		got := digest{
			code:        price.ProductCode,
			georgian:    price.FuelType,
			price:       price.Price.String(),
			selfService: price.PriceSelfService.String(),
			date:        price.Date,
			active:      price.IsActive,
		}
		if got != want {
			t.Errorf("%s = %+v, want %+v", name, got, want)
		}
	}
}

// WISFUEL-SC-002: A status other than 200 is an APIError with the first 1 KiB of the body.
func TestFuelPricesReportsAStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "short body", body: "error", want: "error"},
		{name: "long body", body: strings.Repeat("x", 2048), want: strings.Repeat("x", 1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(tt.body))
				}),
			)
			t.Cleanup(server.Close)

			prices, err := wissol.New(vendorhttp.WithBaseURL(server.URL)).FuelPrices(t.Context())
			if prices != nil {
				t.Errorf("FuelPrices() = %+v, want nil", prices)
			}

			apiErr, ok := errors.AsType[*wissol.APIError](err)
			if !ok {
				t.Fatalf("FuelPrices() error = %v, want an *APIError", err)
			}

			if apiErr.StatusCode != http.StatusInternalServerError || apiErr.Body != tt.want {
				t.Errorf("APIError = %d, %q", apiErr.StatusCode, apiErr.Body)
			}

			if !errors.Is(err, wissol.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(%v, ErrUnexpectedStatus) = false", err)
			}
		})
	}
}

// WISFUEL-SC-002: A body that is not the list of products is an error.
func TestFuelPricesReportsABody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html></html>`))
	}))
	t.Cleanup(server.Close)

	if _, err := wissol.New(vendorhttp.WithBaseURL(server.URL)).
		FuelPrices(t.Context()); err == nil {
		t.Error("FuelPrices() error = nil, want a decode error")
	}
}
