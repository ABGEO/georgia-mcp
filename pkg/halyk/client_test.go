package halyk_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/halyk"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	recordedPage = "testdata/currency-ka.html"
	dollar       = "USD"
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
	start := strings.LastIndex(page[:at], "<tr>")
	end := at + strings.Index(page[at:], "</tr>")

	if at < 0 || start < 0 || end < at {
		t.Fatal("the page holds no row of USD")
	}

	return page[:start] + edit(page[start:end]) + page[end:]
}

func serve(t *testing.T, status int, body string) *halyk.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return halyk.New(vendorhttp.WithBaseURL(server.URL))
}

// HALRATE-SC-001: The client asks for the Georgian rate page as HTML, with the
// User-Agent of the repository and no key.
func TestRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write([]byte(recorded(t)))
	}))
	t.Cleanup(server.Close)

	if _, err := halyk.New(vendorhttp.WithBaseURL(server.URL)).Rates(t.Context()); err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if got.URL.Path != "/ka/individuals/currency" {
		t.Errorf("path = %q, want /ka/individuals/currency", got.URL.Path)
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

// HALRATE-SC-002: The client reads 7 rows from the recorded page, and no row of GEL.
func TestRatesReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	rows, err := serve(t, http.StatusOK, recorded(t)).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(rows) != 7 {
		t.Fatalf("Rates() = %d rows, want 7", len(rows))
	}

	byCode := make(map[string]halyk.RateRow, len(rows))
	for _, row := range rows {
		byCode[row.Currency] = row
	}

	if _, ok := byCode["GEL"]; ok {
		t.Error("Rates() holds a row of GEL, want none")
	}

	for _, want := range []halyk.RateRow{
		{Currency: dollar, Buy: "2.5870", Sell: "2.6120"},
		{Currency: "KZT", Buy: "0.0054", Sell: "0.0059"},
	} {
		if got := byCode[want.Currency]; got != want {
			t.Errorf("row of %s = %+v, want %+v", want.Currency, got, want)
		}
	}
}

// HALRATE-SC-003: A page with no current tab, a renamed title, or a short row is an error.
func TestRatesReportsABrokenPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no current tab",
			body: strings.ReplaceAll(recorded(t), `id="currentexchangerate"`, `id="other"`),
			want: halyk.ErrNoRateTable,
		},
		{
			name: "sell title renamed",
			body: strings.Replace(recorded(t), "<th>გაყიდვა</th>", "<th>ფასი</th>", 1),
			want: halyk.ErrMalformedTable,
		},
		{
			name: "row of two cells",
			body: withDollarRow(t, func(row string) string {
				kept, _, _ := strings.CutLast(row, "<td>")

				return kept
			}),
			want: halyk.ErrMalformedTable,
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

// HALRATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusServiceUnavailable, strings.Repeat("x", 2048)).Rates(t.Context())

	apiErr, ok := errors.AsType[*halyk.APIError](err)
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

	if !errors.Is(err, halyk.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
