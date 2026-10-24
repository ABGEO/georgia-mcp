package gulf_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/gulf"
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

const euroDiesel = "Euro Diesel"

const head = `<thead><tr class="prices_header">` +
	`<td class="first"><a><span class="normal">Date</span><span class="small"></span></a></td>` +
	`<td class="g_col"><a><span class="normal">Euro Diesel</span><span class="small">ED</span></a></td>` +
	`<td class="gp_col"> Euro Diesel </td>` +
	`</tr></thead>`

// GULFUEL-SC-001: The client sends the path, the Accept header, and the User-Agent.
func TestPriceArchiveSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`<table id="price_archive_tbl">` + head + `</table>`))
	}))
	t.Cleanup(server.Close)

	if _, err := gulf.New(vendorhttp.WithBaseURL(server.URL)).
		PriceArchive(t.Context()); err != nil {
		t.Fatalf("PriceArchive() error = %v", err)
	}

	if got.URL.Path != "/en/fuel_prices" {
		t.Errorf("path = %q, want /en/fuel_prices", got.URL.Path)
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

// GULFUEL-SC-001: The client returns each column with its class, and each row of a
// recorded page, as the page shows them.
func TestPriceArchiveReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("testdata/fuel-prices.html")
	if err != nil {
		t.Fatal(err)
	}

	table, err := gulf.New(vendorhttp.WithBaseURL(serve(t, string(body), http.StatusOK).URL)).
		PriceArchive(t.Context())
	if err != nil {
		t.Fatalf("PriceArchive() error = %v", err)
	}

	if len(table.Columns) != 11 {
		t.Fatalf("Columns holds %d columns, want 11: %+v", len(table.Columns), table.Columns)
	}

	wantColumns := map[int]gulf.Column{
		0: {Name: "Date", Class: "first"},
		1: {Name: "G-Force Super", Class: "g_col"},
		5: {Name: euroDiesel, Class: "g_col"},
		6: {Name: "G-Force Super", Class: "gp_col"},
	}
	for i, want := range wantColumns {
		if got := table.Columns[i]; got != want {
			t.Errorf("Columns[%d] = %+v, want %+v", i, got, want)
		}
	}

	if len(table.Rows) != 100 {
		t.Fatalf("Rows holds %d rows, want 100", len(table.Rows))
	}

	want := []string{
		"2026-10-19",
		"4.69", "4.39", "4.09", "5.05", "4.99",
		"4.59", "4.29", "3.99", "4.95", "4.89",
	}
	if first := table.Rows[0]; !slices.Equal(first, want) {
		t.Errorf("the first row = %q, want %q", first, want)
	}
}

// GULFUEL-SC-002: No price table and a row of the wrong width are errors.
func TestPriceArchiveMirrorsThePage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no table",
			body: `<html><body><p>none</p></body></html>`,
			want: gulf.ErrNoPriceTable,
		},
		{
			name: "a table with another id",
			body: `<table id="other">` + head + `</table>`,
			want: gulf.ErrNoPriceTable,
		},
		{
			name: "a row of the wrong width",
			body: `<table id="price_archive_tbl">` + head +
				`<tbody><tr><td>2026-10-19</td><td>4.99</td></tr></tbody></table>`,
			want: gulf.ErrMalformedTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := gulf.New(vendorhttp.WithBaseURL(serve(t, tt.body, http.StatusOK).URL)).
				PriceArchive(t.Context())
			if !errors.Is(err, tt.want) {
				t.Errorf("PriceArchive() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// GULFUEL-SC-002: A head with no row gives each column and no row, with no error.
func TestPriceArchiveReadsAHeadWithNoRow(t *testing.T) {
	t.Parallel()

	empty := `<table id="price_archive_tbl">` + head + `<tbody></tbody></table>`

	table, err := gulf.New(vendorhttp.WithBaseURL(serve(t, empty, http.StatusOK).URL)).
		PriceArchive(t.Context())
	if err != nil || len(table.Columns) != 3 || len(table.Rows) != 0 {
		t.Errorf("PriceArchive() = %+v, %v, want a head of 3 and no row", table, err)
	}

	wantNames := []string{"Date", euroDiesel, euroDiesel}
	if names := namesOf(table.Columns); !slices.Equal(names, wantNames) {
		t.Errorf("names = %q, want %q", names, wantNames)
	}
}

// GULFUEL-SC-002: A status other than 200 is an APIError that wraps ErrUnexpectedStatus.
func TestPriceArchiveReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := gulf.New(vendorhttp.WithBaseURL(serve(t, "refused", http.StatusForbidden).URL)).
		PriceArchive(t.Context())

	apiErr, ok := errors.AsType[*gulf.APIError](err)
	if !ok || apiErr.StatusCode != http.StatusForbidden || apiErr.Body != "refused" {
		t.Fatalf("PriceArchive() error = %v, want an *APIError with 403 and the body", err)
	}

	if !errors.Is(err, gulf.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}

func namesOf(columns []gulf.Column) []string {
	names := make([]string, 0, len(columns))

	for _, column := range columns {
		names = append(names, column.Name)
	}

	return names
}
