package scapp_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/scapp"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	recordedPage = "testdata/currency-ka.html"
	dollarUnit   = "1 USD"
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
	at := strings.Index(page, "usd-flag.svg")
	start := strings.LastIndex(page[:at], "<tr")
	end := at + strings.Index(page[at:], "</tr>")

	if at < 0 || start < 0 || end < at {
		t.Fatal("the page holds no row of USD")
	}

	return page[:start] + edit(page[start:end]) + page[end:]
}

func serve(t *testing.T, status int, body string) *scapp.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return scapp.New(vendorhttp.WithBaseURL(server.URL))
}

// SCAPRATE-SC-001: The client asks for the Georgian rate page as HTML, with the
// User-Agent of the repository and no key.
func TestRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write([]byte(recorded(t)))
	}))
	t.Cleanup(server.Close)

	if _, err := scapp.New(vendorhttp.WithBaseURL(server.URL)).Rates(t.Context()); err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if got.URL.Path != "/ka/currency" {
		t.Errorf("path = %q, want /ka/currency", got.URL.Path)
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

// SCAPRATE-SC-002: The client reads 6 rows from the recorded page, with the unit and
// the code in one cell.
func TestRatesReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	rows, err := serve(t, http.StatusOK, recorded(t)).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(rows) != 6 {
		t.Fatalf("Rates() = %d rows, want 6", len(rows))
	}

	byUnit := make(map[string]scapp.RateRow, len(rows))
	for _, row := range rows {
		byUnit[row.Unit] = row
	}

	for _, want := range []scapp.RateRow{
		{Unit: dollarUnit, Buy: "2.595", Sell: "2.599"},
		{Unit: "1 EUR", Buy: "2.900", Sell: "2.914"},
	} {
		if got := byUnit[want.Unit]; got != want {
			t.Errorf("row of %s = %+v, want %+v", want.Unit, got, want)
		}
	}
}

// SCAPRATE-SC-003: A page with no table, a renamed title, or a short row is an error.
func TestRatesReportsABrokenPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no table",
			body: "<html><body><p>Scapp</p></body></html>",
			want: scapp.ErrNoRateTable,
		},
		{
			name: "buy title renamed",
			body: strings.Replace(recorded(t), "<span>ყიდვა</span>", "<span>ფასი</span>", 1),
			want: scapp.ErrMalformedTable,
		},
		{
			name: "row of two cells",
			body: withDollarRow(t, func(row string) string {
				kept, _, _ := strings.CutLast(row, "<td")

				return kept
			}),
			want: scapp.ErrMalformedTable,
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

// SCAPRATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusServiceUnavailable, strings.Repeat("x", 2048)).Rates(t.Context())

	apiErr, ok := errors.AsType[*scapp.APIError](err)
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

	if !errors.Is(err, scapp.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
