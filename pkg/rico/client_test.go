package rico_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/rico"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	recordedPage = "testdata/home-ka.html"
	dollar       = "USD"
	sellLabel    = "<span>&#x10D2;&#x10D0;&#x10E7;&#x10D8;&#x10D3;&#x10D5;&#x10D0;</span>"
)

func recorded(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile(recordedPage)
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// withDollarRow returns the page with the row of USD changed by edit.
func withDollarRow(t *testing.T, edit func(row string) string) string {
	t.Helper()

	page := recorded(t)
	at := strings.Index(page, `alt="USD"`)
	start := strings.LastIndex(page[:at], `<div class="currency-row">`)
	end := at + strings.Index(page[at:], `<div class="currency-row">`)

	if at < 0 || start < 0 || end < at {
		t.Fatal("the page holds no row of USD")
	}

	return page[:start] + edit(page[start:end]) + page[end:]
}

func serve(t *testing.T, status int, body string) *rico.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return rico.New(vendorhttp.WithBaseURL(server.URL))
}

// RICORATE-SC-001: The client asks for the Georgian home page as HTML, with the
// User-Agent of the repository and no key.
func TestRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(recorded(t)))
	}))
	t.Cleanup(server.Close)

	if _, err := rico.New(vendorhttp.WithBaseURL(server.URL)).Rates(t.Context()); err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if got.URL.Path != "/ka/" {
		t.Errorf("path = %q, want /ka/", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-rico/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-rico/<version> (+<repository>)", agent)
	}

	for _, name := range []string{"Apikey", "Authorization"} {
		if value := got.Header.Get(name); value != "" {
			t.Errorf("%s = %q, want no key header", name, value)
		}
	}
}

// RICORATE-SC-002: The client reads 13 rows from the recorded page, the cross rate
// included.
func TestRatesReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	rows, err := serve(t, http.StatusOK, recorded(t)).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(rows) != 13 {
		t.Fatalf("Rates() = %d rows, want 13", len(rows))
	}

	byCode := make(map[string]rico.RateRow, len(rows))
	for _, row := range rows {
		byCode[row.Currency] = row
	}

	for _, want := range []rico.RateRow{
		{Currency: dollar, Unit: "1", Buy: "2.5960", Sell: "2.6000"},
		{Currency: "USD-EUR", Unit: "1", Buy: "1.1204", Sell: "1.1260"},
		{Currency: "PLN", Unit: "10", Buy: "6.1000", Sell: "6.7000"},
	} {
		if got := byCode[want.Currency]; got != want {
			t.Errorf("row of %s = %+v, want %+v", want.Currency, got, want)
		}
	}
}

// RICORATE-SC-003: The label of a price, not its place, tells buy from sell. A page with
// no table or a row with no sell label is an error. A status that is not 200 is an
// APIError.
func TestRatesReadsEachPriceByItsLabel(t *testing.T) {
	t.Parallel()

	swapped := withDollarRow(t, func(row string) string {
		first := strings.Index(row, `<div class="currency-rate">`)
		second := strings.LastIndex(row, `<div class="currency-rate">`)
		closing := strings.LastIndex(row, "</div>")

		return row[:first] + row[second:closing] + row[first:second] + row[closing:]
	})

	rows, err := serve(t, http.StatusOK, swapped).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	want := rico.RateRow{Currency: dollar, Unit: "1", Buy: "2.5960", Sell: "2.6000"}
	if rows[0] != want {
		t.Errorf("row of USD = %+v, want %+v", rows[0], want)
	}
}

// RICORATE-SC-003: A page with no table, or a row with no sell label, is an error.
func TestRatesReportsABrokenPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no table",
			body: "<html><body><p>Rico</p></body></html>",
			want: rico.ErrNoRateTable,
		},
		{
			name: "no sell label",
			body: withDollarRow(t, func(row string) string {
				return strings.Replace(row, sellLabel, "<span></span>", 1)
			}),
			want: rico.ErrMalformedTable,
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

// RICORATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusServiceUnavailable, strings.Repeat("x", 2048)).Rates(t.Context())

	apiErr, ok := errors.AsType[*rico.APIError](err)
	if !ok {
		t.Fatalf("Rates() error = %v, want an APIError", err)
	}

	want := rico.APIError{
		StatusCode: http.StatusServiceUnavailable,
		Body:       strings.Repeat("x", 1024),
	}
	if *apiErr != want {
		t.Errorf("APIError = %+v, want %+v", *apiErr, want)
	}

	if !errors.Is(err, rico.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}

func TestAPIErrorText(t *testing.T) {
	t.Parallel()

	err := &rico.APIError{StatusCode: http.StatusBadGateway}
	if got, want := err.Error(), "rico: unexpected status: 502: Bad Gateway"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
