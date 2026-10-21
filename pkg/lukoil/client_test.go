package lukoil_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/lukoil"
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

// LUKFUEL-SC-001: The client sends the path, the Accept header, and the User-Agent.
func TestPriceHistorySendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`<table><thead><tr><th>#</th></tr></thead></table>`))
	}))
	t.Cleanup(server.Close)

	if _, err := lukoil.New(lukoil.WithBaseURL(server.URL)).PriceHistory(t.Context()); err != nil {
		t.Fatalf("PriceHistory() error = %v", err)
	}

	if got.URL.Path != "/prices-history" {
		t.Errorf("path = %q, want /prices-history", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-lukoil/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-lukoil/<version> (+<repository>)", agent)
	}
}

// LUKFUEL-SC-001: The client returns the head and each row of a recorded page, as the
// page shows them.
func TestPriceHistoryReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/prices-history.html")
	if err != nil {
		t.Fatal(err)
	}

	table, err := lukoil.New(lukoil.WithBaseURL(serve(t, string(body), http.StatusOK).URL)).
		PriceHistory(t.Context())
	if err != nil {
		t.Fatalf("PriceHistory() error = %v", err)
	}

	wantHeader := []string{
		"#", "Super Ecto 100", "Super Ecto", "Premium Avangard", "Euro Regular", "Euro Diesel",
		"შეცვლის თარიღი",
	}
	if !slices.Equal(table.Header, wantHeader) {
		t.Errorf("Header = %q, want %q", table.Header, wantHeader)
	}

	if len(table.Rows) != 106 {
		t.Fatalf("Rows holds %d rows, want 106", len(table.Rows))
	}

	want := []string{"106", "0.00", "4.60", "4.25", "3.95", "4.97", "2026-10-05 10:01:46"}
	if last := table.Rows[len(table.Rows)-1]; !slices.Equal(last, want) {
		t.Errorf("the last row = %q, want %q", last, want)
	}
}

// LUKFUEL-SC-002: The client mirrors the page: no table and a row of the wrong width are
// errors, a head with no row is not, and a status other than 200 is an APIError.
func TestPriceHistoryMirrorsThePage(t *testing.T) {
	t.Parallel()

	head := `<thead><tr><th>#</th><th>Euro Diesel</th><th>date</th></tr></thead>`

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no table",
			body: `<html><body><p>none</p></body></html>`,
			want: lukoil.ErrNoPriceTable,
		},
		{
			name: "a row of the wrong width",
			body: `<table>` + head + `<tbody><tr><td>1</td><td>4.97</td></tr></tbody></table>`,
			want: lukoil.ErrMalformedTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := lukoil.New(lukoil.WithBaseURL(serve(t, tt.body, http.StatusOK).URL)).
				PriceHistory(t.Context())
			if !errors.Is(err, tt.want) {
				t.Errorf("PriceHistory() error = %v, want %v", err, tt.want)
			}
		})
	}

	empty := `<table>` + head + `<tbody></tbody></table>`

	table, err := lukoil.New(lukoil.WithBaseURL(serve(t, empty, http.StatusOK).URL)).
		PriceHistory(t.Context())
	if err != nil || len(table.Header) != 3 || len(table.Rows) != 0 {
		t.Errorf("PriceHistory() = %+v, %v, want a head of 3 and no row", table, err)
	}

	_, err = lukoil.New(lukoil.WithBaseURL(serve(t, "down", http.StatusInternalServerError).URL)).
		PriceHistory(t.Context())

	apiErr, ok := errors.AsType[*lukoil.APIError](err)
	if !ok || apiErr.StatusCode != http.StatusInternalServerError || apiErr.Body != "down" {
		t.Fatalf("PriceHistory() error = %v, want an *APIError with 500 and the body", err)
	}

	if !errors.Is(err, lukoil.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
