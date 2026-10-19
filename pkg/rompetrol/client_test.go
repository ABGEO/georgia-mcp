package rompetrol_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/rompetrol"
)

func serve(t *testing.T, body []byte) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server
}

func serveFile(t *testing.T, path string) *httptest.Server {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return serve(t, body)
}

// page returns a short page with the anchor of the price table and the given table.
func page(table string) []byte {
	return []byte(`<!DOCTYPE html><html><body><a id="pricelist"></a><section>` +
		table + `</section></body></html>`)
}

// ROMFUEL-SC-001: The client sends the path, the Accept header, and the User-Agent.
func TestPricesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write(page(`<table><tbody></tbody></table>`))
	}))
	t.Cleanup(server.Close)

	if _, err := rompetrol.New(rompetrol.WithBaseURL(server.URL)).Prices(t.Context()); err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	if got.URL.Path != "/" {
		t.Errorf("path = %q, want /", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-rompetrol/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-rompetrol/<version> (+<repository>)", agent)
	}
}

// ROMFUEL-SC-001: The client reads the 10 rows of a recorded page, as the page shows them.
func TestPricesReadsARecordedPage(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/home.html")

	got, err := rompetrol.New(rompetrol.WithBaseURL(server.URL)).Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	want := []rompetrol.PriceRow{
		{Product: "efix სუპერი", Price: "4.65"},
		{Product: "- efix სუპერი (თვითმომსახურების სადგურებზე)", Price: "4.55"},
		{Product: "efix ევრო პრემიუმი", Price: "4.32"},
		{Product: "- efix ევრო პრემიუმი (თვითმომსახურების სადგურებზე)", Price: "4.22"},
		{Product: "efix ევრო რეგულარი", Price: "4.02"},
		{Product: "- ევრო რეგულარი (თვითმომსახურების სადგურებზე)", Price: "3.92"},
		{Product: "efix ევრო დიზელი", Price: "4.99"},
		{Product: "- efix ევრო დიზელი (თვითმომსახურების სადგურებზე)", Price: "4.89"},
		{Product: "ევრო დიზელი", Price: "4.92"},
		{Product: "- ევრო დიზელი (თვითმომსახურების სადგურებზე)", Price: "4.82"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Prices() = %q, want %q", got, want)
	}
}

// ROMFUEL-SC-002: A status other than 200 is an APIError with the first 1 KiB of the body.
func TestPricesReportsAStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		wantBody string
	}{
		{name: "short body", body: "error", wantBody: "error"},
		{
			name:     "long body",
			body:     strings.Repeat("x", 2048),
			wantBody: strings.Repeat("x", 1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(tt.body))
				}),
			)
			t.Cleanup(server.Close)

			_, err := rompetrol.New(rompetrol.WithBaseURL(server.URL)).Prices(t.Context())

			var apiErr *rompetrol.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("Prices() error = %v, want *APIError", err)
			}

			if apiErr.StatusCode != http.StatusServiceUnavailable {
				t.Errorf("StatusCode = %d, want 503", apiErr.StatusCode)
			}

			if apiErr.Body != tt.wantBody {
				t.Errorf("Body has %d bytes, want %d", len(apiErr.Body), len(tt.wantBody))
			}

			if !errors.Is(err, rompetrol.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
			}
		})
	}
}

// ROMFUEL-SC-003: A page with no table, or with a malformed table, is an error. A table
// with a head only is an empty list.
func TestPricesReportsTheShapeOfTheTable(t *testing.T) {
	t.Parallel()

	recorded, err := os.ReadFile("testdata/home-no-table.html")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		body    []byte
		wantErr error
	}{
		{name: "recorded page with no table", body: recorded, wantErr: rompetrol.ErrNoPriceTable},
		{
			name: "anchor with no table after it",
			body: []byte(
				`<html><body><table><tr><td>a</td><td>1</td></tr></table><a id="pricelist"></a></body></html>`,
			),
			wantErr: rompetrol.ErrNoPriceTable,
		},
		{
			name:    "row with 3 cells",
			body:    page(`<table><tr><td>efix სუპერი</td><td>4.65</td><td>x</td></tr></table>`),
			wantErr: rompetrol.ErrMalformedTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := serve(t, tt.body)

			_, err := rompetrol.New(rompetrol.WithBaseURL(server.URL)).Prices(t.Context())
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Prices() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("head only", func(t *testing.T) {
		t.Parallel()

		server := serve(
			t,
			page(`<table><thead><tr><th>პროდუქტი</th><th>ლარი / ლ</th></tr></thead></table>`),
		)

		got, err := rompetrol.New(rompetrol.WithBaseURL(server.URL)).Prices(t.Context())
		if err != nil {
			t.Fatalf("Prices() error = %v, want none", err)
		}

		if len(got) != 0 {
			t.Errorf("Prices() = %v, want an empty list", got)
		}
	})
}
