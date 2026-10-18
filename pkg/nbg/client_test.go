package nbg_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/nbg"
)

const dollar = "USD"

func serveFile(t *testing.T, path string) *httptest.Server {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server
}

func date(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// NBGRATE-SC-022: The client sends the path, the date, the currencies, the Accept header,
// and its User-Agent. With no date and no currencies, the query is empty.
func TestRateSetsSendsTheRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     nbg.RateSetsQuery
		wantQuery string
	}{
		{
			name: "date and currencies",
			query: nbg.RateSetsQuery{
				Date:       date(t, "2026-10-17"),
				Currencies: []string{dollar, "GBP"},
			},
			wantQuery: "currencies=USD%2CGBP&date=2026-10-17",
		},
		{name: "nothing", query: nbg.RateSetsQuery{}, wantQuery: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := recordRequest(t, func(client *nbg.Client) error {
				_, err := client.RateSets(t.Context(), nbg.LanguageGeorgian, tt.query)

				return err
			})

			if got.URL.Path != "/currencies/ka/json/" {
				t.Errorf("path = %q, want %q", got.URL.Path, "/currencies/ka/json/")
			}

			if got.URL.Query().Encode() != tt.wantQuery {
				t.Errorf("query = %q, want %q", got.URL.Query().Encode(), tt.wantQuery)
			}

			if accept := got.Header.Get("Accept"); accept != "application/json" {
				t.Errorf("Accept = %q, want application/json", accept)
			}

			agent := got.Header.Get("User-Agent")
			if !strings.HasPrefix(agent, "georgia-mcp-nbg/") ||
				!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
				t.Errorf("User-Agent = %q, want georgia-mcp-nbg/<version> (+<repository>)", agent)
			}
		})
	}
}

func recordRequest(t *testing.T, send func(*nbg.Client) error) *http.Request {
	t.Helper()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)

	if err := send(nbg.New(nbg.WithBaseURL(server.URL))); err != nil {
		t.Fatalf("request error = %v", err)
	}

	return got
}

func TestRateSetsDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/en-2026-10-17.json")
	client := nbg.New(nbg.WithBaseURL(server.URL))

	sets, err := client.RateSets(
		t.Context(),
		nbg.LanguageEnglish,
		nbg.RateSetsQuery{Date: date(t, "2026-10-17")},
	)
	if err != nil {
		t.Fatalf("RateSets() error = %v", err)
	}

	if len(sets) != 1 || len(sets[0].Currencies) != 42 {
		t.Fatalf("RateSets() = %d sets, want 1 set of 42 currencies", len(sets))
	}

	for _, currency := range sets[0].Currencies {
		if currency.Code == dollar && currency.RateFormated != "2.5982" {
			t.Errorf("USD rateFormated = %q, want %q", currency.RateFormated, "2.5982")
		}
	}
}

// NBGRATE-SC-021: A status that is not a success is an APIError. A body that is not JSON is an error.
func TestRateSetsReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		body       string
		wantStatus bool
	}{
		{
			name:       "server error",
			status:     http.StatusInternalServerError,
			body:       "down",
			wantStatus: true,
		},
		{name: "body that is not JSON", status: http.StatusOK, body: "<html>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
				}),
			)
			t.Cleanup(server.Close)

			_, err := nbg.New(nbg.WithBaseURL(server.URL)).
				RateSets(t.Context(), nbg.LanguageEnglish, nbg.RateSetsQuery{})
			if err == nil {
				t.Fatal("RateSets() error = nil, want an error")
			}

			if !tt.wantStatus {
				return
			}

			apiErr, ok := errors.AsType[*nbg.APIError](err)
			if !ok || apiErr.StatusCode != tt.status || apiErr.Body != tt.body {
				t.Errorf(
					"RateSets() error = %v, want an APIError with %d and %q",
					err,
					tt.status,
					tt.body,
				)
			}

			if !errors.Is(err, nbg.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
			}
		})
	}
}
