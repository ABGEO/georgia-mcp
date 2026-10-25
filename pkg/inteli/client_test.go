package inteli_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/inteli"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	recordedPage = "testdata/main-page-geo.html"
	lari         = "GEL"
	dollar       = "USD"
	title        = "ვალუტის სტანდარტული კურსები"
)

// dollarCells matches the first two cells of the row of USD in GEL.
var dollarCells = regexp.MustCompile(`<td>GEL</td>\s*<td>USD</td>`)

func recorded(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile(recordedPage)
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func serve(t *testing.T, status int, body string) *inteli.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return inteli.New(vendorhttp.WithBaseURL(server.URL))
}

// withoutFirstCell returns the page with the row of USD in GEL cut to three cells.
func withoutFirstCell(t *testing.T) string {
	t.Helper()

	page := recorded(t)

	at := dollarCells.FindStringIndex(page)
	if at == nil {
		t.Fatal("the page holds no row of USD in GEL")
	}

	return page[:at[0]] + "<td>USD</td>" + page[at[1]:]
}

// INTLRATE-SC-001: The client asks for the Georgian home page as HTML, with the
// User-Agent of the repository and no key.
func TestRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write([]byte(recorded(t)))
	}))
	t.Cleanup(server.Close)

	if _, err := inteli.New(vendorhttp.WithBaseURL(server.URL)).Rates(t.Context()); err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if got.URL.Path != "/ka/main-page-geo/" {
		t.Errorf("path = %q, want /ka/main-page-geo/", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}

	for _, name := range []string{"Apikey", "Authorization", "X-Api-Key"} {
		if value := got.Header.Get(name); value != "" {
			t.Errorf("%s = %q, want no key header", name, value)
		}
	}
}

// INTLRATE-SC-002: The client reads 6 rows from the recorded page, the cross rate among
// them, with the cells in the order of the page.
func TestRatesReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	rows, err := serve(t, http.StatusOK, recorded(t)).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(rows) != 6 {
		t.Fatalf("Rates() = %d rows, want 6", len(rows))
	}

	for _, want := range []inteli.RateRow{
		{Base: lari, Currency: dollar, Buy: "2.595", Sell: "2.598"},
		{Base: "EUR", Currency: dollar, Buy: "1.119", Sell: "1.123"},
		{Base: lari, Currency: "TRY", Buy: "0.050", Sell: "0.056"},
	} {
		found := false

		for _, row := range rows {
			if row == want {
				found = true
			}
		}

		if !found {
			t.Errorf("Rates() = %+v, want a row %+v", rows, want)
		}
	}
}

// INTLRATE-SC-003: The client takes the table with the title, not the first table.
func TestRatesFindTheTableByItsTitle(t *testing.T) {
	t.Parallel()

	other := `<table><tbody><tr><td>GEL</td><td>USD</td><td>9</td><td>9</td></tr></tbody></table>`
	page := strings.Replace(
		recorded(t),
		`<div class="rates-col">`,
		other+`<div class="rates-col">`,
		1,
	)

	rows, err := serve(t, http.StatusOK, page).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	want := inteli.RateRow{Base: lari, Currency: dollar, Buy: "2.595", Sell: "2.598"}
	if len(rows) != 6 || rows[0] != want {
		t.Errorf("Rates() = %+v, want 6 rows that start with %+v", rows, want)
	}
}

// INTLRATE-SC-003: A page with no table, a renamed title, or a short row is an error.
func TestRatesReportsABrokenPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no table",
			body: "<html><body><p>Inteli Express</p></body></html>",
			want: inteli.ErrNoRateTable,
		},
		{
			name: "title renamed",
			body: strings.Replace(recorded(t), title, "კურსები", 1),
			want: inteli.ErrNoRateTable,
		},
		{
			name: "row of three cells",
			body: withoutFirstCell(t),
			want: inteli.ErrMalformedTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rows, err := serve(t, http.StatusOK, tt.body).Rates(t.Context())
			if rows != nil || !errors.Is(err, tt.want) {
				t.Errorf("Rates() = %v, %v, want nil and %v", rows, err, tt.want)
			}
		})
	}
}

// INTLRATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusServiceUnavailable, strings.Repeat("x", 2048)).Rates(t.Context())

	apiErr, ok := errors.AsType[*inteli.APIError](err)
	if !ok {
		t.Fatalf("Rates() error = %v, want an APIError", err)
	}

	if apiErr.StatusCode != http.StatusServiceUnavailable ||
		apiErr.Body != strings.Repeat("x", 1024) {
		t.Errorf(
			"APIError = %d with %d bytes, want 503 with 1024",
			apiErr.StatusCode,
			len(apiErr.Body),
		)
	}

	if !errors.Is(err, inteli.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
