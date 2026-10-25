package mjc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/mjc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	recordedPage = "testdata/rates.html"
	dollarUnit   = "1 USD"
)

var (
	// fourCells matches the four cells of one row of the recorded page.
	fourCells = regexp.MustCompile(
		`(<td>\s*<pre>[^<]*</pre>\s*</td>)(\s*<td>\s*<pre>[^<]*</pre>\s*</td>)` +
			`(\s*<td>\s*<pre>[^<]*</pre>\s*</td>)(\s*<td>\s*<pre>[^<]*</pre>\s*</td>)`,
	)
	// fourTitles matches the four titles of the recorded page.
	fourTitles = regexp.MustCompile(`(<th scope="col">[^<]*</th>)(\s*<th scope="col">[^<]*</th>)` +
		`(\s*<th scope="col">[^<]*</th>)(\s*<th scope="col">[^<]*</th>)`)
)

func recorded(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile(recordedPage)
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// withDollarRow returns the page with the row of USD in the table changed by edit.
func withDollarRow(t *testing.T, edit func(row string) string) string {
	t.Helper()

	page := recorded(t)
	at := strings.Index(page, "<pre>"+dollarUnit+"</pre>")
	start := strings.LastIndex(page[:max(at, 0)], "<tr")
	end := at + strings.Index(page[max(at, 0):], "</tr>")

	if at < 0 || start < 0 || end < at {
		t.Fatal("the page holds no row of USD")
	}

	return page[:start] + edit(page[start:end]) + page[end:]
}

func serve(t *testing.T, status int, body string) *mjc.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return mjc.New(vendorhttp.WithBaseURL(server.URL))
}

func byUnit(rows []mjc.RateRow) map[string]mjc.RateRow {
	found := make(map[string]mjc.RateRow, len(rows))
	for _, row := range rows {
		found[row.Unit] = row
	}

	return found
}

// MJCRATE-SC-001: The client asks for the rate page as HTML, with the User-Agent of the
// repository and no key.
func TestRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write([]byte(recorded(t)))
	}))
	t.Cleanup(server.Close)

	if _, err := mjc.New(vendorhttp.WithBaseURL(server.URL)).Rates(t.Context()); err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if got.URL.Path != "/rates" {
		t.Errorf("path = %q, want /rates", got.URL.Path)
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

// MJCRATE-SC-002: The client reads 17 rows from the recorded page, with the unit and the
// code in one cell, and no official rate.
func TestRatesReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	rows, err := serve(t, http.StatusOK, recorded(t)).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(rows) != 17 {
		t.Fatalf("Rates() = %d rows, want 17", len(rows))
	}

	found := byUnit(rows)

	for _, want := range []mjc.RateRow{
		{Unit: dollarUnit, Buy: "2.59500", Sell: "2.59800"},
		{Unit: "1000 AMD", Buy: "7.00000", Sell: "7.20000"},
		{Unit: "1 SAR", Buy: "0.66900", Sell: "0.68500"},
	} {
		if got := found[want.Unit]; got != want {
			t.Errorf("row of %s = %+v, want %+v", want.Unit, got, want)
		}
	}
}

// MJCRATE-SC-003: The client finds each column by its title, so a moved official column
// changes no price.
func TestRatesFindTheColumnsByTitle(t *testing.T) {
	t.Parallel()

	page := fourTitles.ReplaceAllString(recorded(t), "$1$4$2$3")
	page = fourCells.ReplaceAllString(page, "$1$4$2$3")

	if page == recorded(t) {
		t.Fatal("the edit moved no column")
	}

	rows, err := serve(t, http.StatusOK, page).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(rows) != 17 {
		t.Fatalf("Rates() = %d rows, want 17", len(rows))
	}

	want := mjc.RateRow{Unit: dollarUnit, Buy: "2.59500", Sell: "2.59800"}
	if got := byUnit(rows)[dollarUnit]; got != want {
		t.Errorf("row of USD = %+v, want %+v", got, want)
	}
}

// MJCRATE-SC-003: A page with no table, a renamed title, or a short row is an error.
func TestRatesReportsABrokenPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no table",
			body: "<html><body><p>MJC</p></body></html>",
			want: mjc.ErrNoRateTable,
		},
		{
			name: "sell title renamed",
			body: strings.Replace(recorded(t), ">გაყიდვა</th>", ">ფასი</th>", 1),
			want: mjc.ErrMalformedTable,
		},
		{
			name: "row of two cells",
			body: withDollarRow(t, func(row string) string {
				kept, _, _ := strings.Cut(row, "</td>")
				rest := row[len(kept)+len("</td>"):]
				second, _, _ := strings.Cut(rest, "</td>")

				return kept + "</td>" + second + "</td>"
			}),
			want: mjc.ErrMalformedTable,
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

// MJCRATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusServiceUnavailable, strings.Repeat("x", 2048)).Rates(t.Context())

	apiErr, ok := errors.AsType[*mjc.APIError](err)
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

	if !errors.Is(err, mjc.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
