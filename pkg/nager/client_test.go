package nager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/nager"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

type recorded struct {
	path   string
	header http.Header
}

// serve answers each request with the file and the status, and records the request.
func serve(t *testing.T, file string, status int) (*nager.Client, *[]recorded) {
	t.Helper()

	body := []byte("not json")

	if file != "" {
		read, err := os.ReadFile("testdata/" + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		body = read
	}

	var requests []recorded

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, recorded{path: r.URL.Path, header: r.Header.Clone()})

		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return nager.New(vendorhttp.WithBaseURL(server.URL)), &requests
}

// HOLIDAY-SC-001: PublicHolidays sends the year and the country in the path, with the
// Accept header and the User-Agent of the server.
func TestPublicHolidaysSendsTheRequest(t *testing.T) {
	t.Parallel()

	client, requests := serve(t, "ge-2026.json", http.StatusOK)

	if _, err := client.PublicHolidays(t.Context(), 2026, "GE"); err != nil {
		t.Fatalf("PublicHolidays() error = %v", err)
	}

	got := (*requests)[0]
	if got.path != "/api/v3/PublicHolidays/2026/GE" {
		t.Errorf("path = %q, want /api/v3/PublicHolidays/2026/GE", got.path)
	}

	agent := got.header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.Contains(agent, "https://github.com/abgeo/georgia-mcp") {
		t.Errorf("User-Agent = %q, want the agent of Georgia MCP", agent)
	}

	if got.header.Get("Accept") != "application/json" {
		t.Errorf("Accept = %q, want application/json", got.header.Get("Accept"))
	}
}

// HOLIDAY-SC-002: The recorded answer of 2026 gives 17 items, New Year's Day first.
func TestPublicHolidaysReadsTheAnswer(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "ge-2026.json", http.StatusOK)

	holidays, err := client.PublicHolidays(t.Context(), 2026, "GE")
	if err != nil {
		t.Fatalf("PublicHolidays() error = %v", err)
	}

	if len(holidays) != 17 {
		t.Fatalf("len(holidays) = %d, want 17", len(holidays))
	}

	want := nager.PublicHoliday{
		Date:        "2026-01-01",
		LocalName:   "ახალი წელი",
		Name:        "New Year's Day",
		CountryCode: "GE",
		Global:      true,
		Types:       []string{"Public"},
	}
	if !reflect.DeepEqual(holidays[0], want) {
		t.Errorf("holidays[0] = %+v, want %+v", holidays[0], want)
	}
}

// HOLIDAY-SC-003: A status that is not a success is an *APIError, and a body that is not
// JSON is an error.
func TestPublicHolidaysFails(t *testing.T) {
	t.Parallel()

	t.Run("status", func(t *testing.T) {
		t.Parallel()

		client, _ := serve(t, "year-400.json", http.StatusBadRequest)

		_, err := client.PublicHolidays(t.Context(), 1974, "GE")

		var apiErr *nager.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
			t.Fatalf("error = %v, want an *APIError with 400", err)
		}

		if !errors.Is(err, nager.ErrUnexpectedStatus) {
			t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false, want true")
		}
	})

	t.Run("body", func(t *testing.T) {
		t.Parallel()

		client, _ := serve(t, "", http.StatusOK)

		if _, err := client.PublicHolidays(t.Context(), 2026, "GE"); err == nil {
			t.Fatal("error = nil, want an error")
		}
	})
}
