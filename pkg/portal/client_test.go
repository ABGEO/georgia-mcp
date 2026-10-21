package portal_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/portal"
)

func serve(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/x-component")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// PORFUEL-SC-001: The client sends the path and the headers of the data stream.
func TestFuelPricesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`6:{"fuelPricesData":[]}`))
	}))
	t.Cleanup(server.Close)

	if _, err := portal.New(portal.WithBaseURL(server.URL)).FuelPrices(t.Context()); err != nil {
		t.Fatalf("FuelPrices() error = %v", err)
	}

	if got.URL.Path != "/ka/fuel-prices" {
		t.Errorf("path = %q, want /ka/fuel-prices", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "text/x-component" {
		t.Errorf("Accept = %q, want text/x-component", accept)
	}

	if rsc := got.Header.Get("Rsc"); rsc != "1" {
		t.Errorf("RSC = %q, want 1", rsc)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-portal/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-portal/<version> (+<repository>)", agent)
	}
}

// PORFUEL-SC-001: The client decodes the price list of a recorded stream.
func TestFuelPricesDecodesTheRecordedStream(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/fuel-prices.rsc")
	if err != nil {
		t.Fatal(err)
	}

	prices, err := portal.New(portal.WithBaseURL(serve(t, string(body), http.StatusOK).URL)).
		FuelPrices(t.Context())
	if err != nil {
		t.Fatalf("FuelPrices() error = %v", err)
	}

	if len(prices) != 5 {
		t.Fatalf("FuelPrices() holds %d entries, want 5", len(prices))
	}

	super := prices[0]
	if super.FuelType.Slug != "super" || super.LatestPrice != "4.39" ||
		super.LatestOnlinePrice != "4.19" {
		t.Errorf("the first entry = %+v, want super at 4.39, online 4.19", super)
	}
}

// PORFUEL-SC-002: The client mirrors the stream: no list and a list that does not decode
// are errors, an empty list is not, and a status other than 200 is an APIError.
func TestFuelPricesMirrorsTheStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{name: "no list", body: `0:{"children":"x"}`, want: portal.ErrNoPriceList},
		{
			name: "malformed",
			body: `6:{"fuelPricesData":[{"name":`,
			want: portal.ErrMalformedPriceList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prices, err := portal.New(portal.WithBaseURL(serve(t, tt.body, http.StatusOK).URL)).
				FuelPrices(t.Context())
			if !errors.Is(err, tt.want) || prices != nil {
				t.Errorf("FuelPrices() = %v, %v, want %v", prices, err, tt.want)
			}
		})
	}

	prices, err := portal.New(portal.WithBaseURL(serve(t, `6:{"fuelPricesData":[]}`, 200).URL)).
		FuelPrices(t.Context())
	if err != nil || len(prices) != 0 {
		t.Errorf("FuelPrices() = %v, %v, want an empty list and no error", prices, err)
	}

	_, err = portal.New(portal.WithBaseURL(serve(t, "down", http.StatusInternalServerError).URL)).
		FuelPrices(t.Context())

	apiErr, ok := errors.AsType[*portal.APIError](err)
	if !ok || apiErr.StatusCode != http.StatusInternalServerError || apiErr.Body != "down" {
		t.Fatalf("FuelPrices() error = %v, want an *APIError with 500 and the body", err)
	}

	if !errors.Is(err, portal.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
