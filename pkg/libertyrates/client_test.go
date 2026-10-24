package libertyrates_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/libertyrates"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	recordedPage = "testdata/home-ka.html"
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

func serve(t *testing.T, status int, body string) *libertyrates.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return libertyrates.New(vendorhttp.WithBaseURL(server.URL))
}

// LIBRATE-SC-001: The client asks for the Georgian home page as HTML, with the
// User-Agent of the repository and no key.
func TestRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(recorded(t)))
	}))
	t.Cleanup(server.Close)

	if _, err := libertyrates.New(vendorhttp.WithBaseURL(server.URL)).
		Rates(t.Context()); err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if got.URL.Path != "/ka/" {
		t.Errorf("path = %q, want /ka/", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}

	for _, name := range []string{"Apikey", "Authorization"} {
		if value := got.Header.Get(name); value != "" {
			t.Errorf("%s = %q, want no key header", name, value)
		}
	}
}

// LIBRATE-SC-002: The client reads 8 rows from the recorded page, each column by its
// title.
func TestRatesReadsTheRecordedPage(t *testing.T) {
	t.Parallel()

	rows, err := serve(t, http.StatusOK, recorded(t)).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(rows) != 8 {
		t.Fatalf("Rates() = %d rows, want 8", len(rows))
	}

	byCode := make(map[string]libertyrates.RateRow, len(rows))
	for _, row := range rows {
		byCode[row.Currency] = row
	}

	want := map[string]libertyrates.RateRow{
		dollar: {
			Currency:     dollar,
			Unit:         "1 USD",
			Official:     "2.5978",
			Commercial:   libertyrates.Quote{Buy: "2.5470", Sell: "2.6470"},
			InternetBank: libertyrates.Quote{Buy: "2.5870", Sell: "2.6070"},
			Transfer:     libertyrates.Quote{Buy: "2.5960", Sell: "2.6000"},
		},
		"AMD": {
			Currency:   "AMD",
			Unit:       "1000 AMD",
			Official:   "7.1737",
			Commercial: libertyrates.Quote{Buy: "6.2220", Sell: "7.5730"},
		},
	}

	for code, row := range want {
		if got := byCode[code]; got != row {
			t.Errorf("row of %s = %+v, want %+v", code, got, row)
		}
	}
}

// LIBRATE-SC-003: A page with no block, with no commercial title, or with a column of
// one price is an error. A status that is not 200 is an APIError.
func TestRatesReportsABrokenPage(t *testing.T) {
	t.Parallel()

	page := recorded(t)

	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "no block",
			body: "<html><body><p>Liberty</p></body></html>",
			want: libertyrates.ErrNoRateBlock,
		},
		{
			name: "no commercial title",
			body: strings.ReplaceAll(page, "კომერციული", "კომერცია"),
			want: libertyrates.ErrMalformedBlock,
		},
		{
			name: "one price in a column",
			body: strings.Replace(page,
				`<span class="currency-rates__currency caps bold">2.6470</span>`, "", 1),
			want: libertyrates.ErrMalformedBlock,
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

// LIBRATE-SC-003: A 503 is an APIError with the first 1 KiB of the body.
func TestRatesReportsAStatus(t *testing.T) {
	t.Parallel()

	_, err := serve(t, http.StatusServiceUnavailable, strings.Repeat("x", 2048)).
		Rates(t.Context())

	apiErr, ok := errors.AsType[*libertyrates.APIError](err)
	if !ok {
		t.Fatalf("Rates() error = %v, want an APIError", err)
	}

	want := libertyrates.APIError{
		StatusCode: http.StatusServiceUnavailable,
		Body:       strings.Repeat("x", 1024),
	}
	if *apiErr != want {
		t.Errorf("APIError = %+v, want %+v", *apiErr, want)
	}

	if !errors.Is(err, libertyrates.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}

func TestAPIErrorText(t *testing.T) {
	t.Parallel()

	err := &libertyrates.APIError{StatusCode: http.StatusBadGateway}
	if got, want := err.Error(), "libertyrates: unexpected status: 502: Bad Gateway"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
