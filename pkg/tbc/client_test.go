package tbc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/tbc"
)

const (
	dollar = "USD"
	euro   = "EUR"
)

func serveFile(t *testing.T, status int, path string) *httptest.Server {
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

// TBCRATE-SC-001: The client sends the path, the currencies joined with commas, the key
// in the header apikey, and the Accept header. With no currencies, the query is empty.
func TestCommercialRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		currencies []string
		wantQuery  string
	}{
		{name: "currencies", currencies: []string{dollar, euro}, wantQuery: "currency=USD%2CEUR"},
		{name: "nothing", currencies: nil, wantQuery: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got *http.Request

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = r.Clone(r.Context())

					_, _ = w.Write([]byte(`{"base":"GEL","commercialRatesList":[]}`))
				}),
			)
			t.Cleanup(server.Close)

			client := tbc.New(tbc.WithBaseURL(server.URL), tbc.WithAPIKey("test-key"))
			if _, err := client.CommercialRates(t.Context(), tt.currencies); err != nil {
				t.Fatalf("CommercialRates() error = %v", err)
			}

			if got.URL.Path != "/v1/exchange-rates/commercial" {
				t.Errorf("path = %q, want %q", got.URL.Path, "/v1/exchange-rates/commercial")
			}

			if got.URL.RawQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", got.URL.RawQuery, tt.wantQuery)
			}

			if key := got.Header.Get("Apikey"); key != "test-key" {
				t.Errorf("Apikey = %q, want %q", key, "test-key")
			}

			if accept := got.Header.Get("Accept"); accept != "application/json" {
				t.Errorf("Accept = %q, want application/json", accept)
			}

			if agent := got.Header.Get(
				"User-Agent",
			); !strings.HasPrefix(
				agent,
				"georgia-mcp-tbc/",
			) {
				t.Errorf("User-Agent = %q, want the prefix georgia-mcp-tbc/", agent)
			}
		})
	}
}

// TBCRATE-SC-002: The client decodes a recorded answer of TBC.
func TestCommercialRatesDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	server := serveFile(t, http.StatusOK, "testdata/commercial.json")

	got, err := tbc.New(tbc.WithBaseURL(server.URL)).
		CommercialRates(t.Context(), []string{dollar, euro, "GBP", "JPY", "XXX"})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	want := &tbc.CommercialRates{
		Base: "GEL",
		CommercialRatesList: []tbc.CommercialRate{
			{Currency: "JPY", Buy: "0.0157", Sell: "0.0172"},
			{Currency: "USD", Buy: "2.566", Sell: "2.628"},
			{Currency: "GBP", Buy: "3.377", Sell: "3.537"},
			{Currency: "EUR", Buy: "2.88", Sell: "2.959"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CommercialRates() = %+v, want %+v", got, want)
	}
}

// TBCRATE-SC-003: A 401 in the problem format is an APIError with its title and system code.
func TestCommercialRatesReportsAProblem(t *testing.T) {
	t.Parallel()

	server := serveFile(t, http.StatusUnauthorized, "testdata/problem-401.json")

	_, err := tbc.New(tbc.WithBaseURL(server.URL)).CommercialRates(t.Context(), nil)

	apiErr, ok := errors.AsType[*tbc.APIError](err)
	if !ok {
		t.Fatalf("CommercialRates() error = %v, want an APIError", err)
	}

	want := &tbc.APIError{
		StatusCode: http.StatusUnauthorized,
		Type:       "https://developers.tbcbank.ge/docs/error-type-401",
		Title:      "Invalid Api Key",
		Detail:     "Please make sure api key is valid",
		SystemCode: "System code for problem is exchange-rates.401.001",
	}
	if *apiErr != *want {
		t.Errorf("APIError = %+v, want %+v", apiErr, want)
	}

	if !errors.Is(err, tbc.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}

func TestAPIErrorWithABodyThatIsNotAProblem(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>"))
	}))
	t.Cleanup(server.Close)

	_, err := tbc.New(tbc.WithBaseURL(server.URL)).CommercialRates(t.Context(), nil)

	want := "tbc: unexpected status: 502: Bad Gateway"
	if err == nil || !strings.HasSuffix(err.Error(), want) {
		t.Errorf("CommercialRates() error = %v, want the suffix %q", err, want)
	}
}
