package gse_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/gse"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	gridPath      = "/storage/uploads/fact/GSE_EXPORT_NEW_PERIOD_03M_16112026_1200.XML"
	frequencyPath = "/storage/uploads/fact/GSE_EXPORT_PERIOD_01M_16112026_1200.XML"
)

type recorded struct {
	path   string
	header http.Header
}

// serve answers each request with the file and the status, and records the request.
func serve(t *testing.T, file string, status int) (*gse.Client, *[]recorded) {
	t.Helper()

	body := []byte("not xml")

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

	return gse.New(vendorhttp.WithBaseURL(server.URL)), &requests
}

func noon(t *testing.T) time.Time {
	t.Helper()

	zone, err := time.LoadLocation("Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}

	return time.Date(2026, time.November, 16, 12, 0, 0, 0, zone)
}

// GSEGRID-SC-001: Grid and Frequency name the file with the time in its own location, and
// send the User-Agent of the server.
func TestClientSendsTheFileName(t *testing.T) {
	t.Parallel()

	client, requests := serve(t, "grid-16112026-1200.xml", http.StatusOK)

	if _, err := client.Grid(t.Context(), noon(t)); err != nil {
		t.Fatalf("Grid() error = %v", err)
	}

	if _, err := client.Frequency(t.Context(), noon(t)); err != nil {
		t.Fatalf("Frequency() error = %v", err)
	}

	for i, want := range []string{gridPath, frequencyPath} {
		got := (*requests)[i]
		if got.path != want {
			t.Errorf("path = %q, want %q", got.path, want)
		}

		agent := got.header.Get("User-Agent")
		if !strings.HasPrefix(agent, "georgia-mcp/") ||
			!strings.Contains(agent, "https://github.com/abgeo/georgia-mcp") {
			t.Errorf("User-Agent = %q, want georgia-mcp/ and the repository", agent)
		}
	}
}

// GSEGRID-SC-002: The grid file decodes as published.
func TestClientDecodesTheGrid(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "grid-16112026-1200.xml", http.StatusOK)

	grid, err := client.Grid(t.Context(), noon(t))
	if err != nil {
		t.Fatalf("Grid() error = %v", err)
	}

	if grid.Header.Time != "16.11.2026 12:00:00" || grid.Header.Period != "03M" {
		t.Errorf("Header = %+v, want 16.11.2026 12:00:00 and 03M", grid.Header)
	}

	if len(grid.Blocks) != 10 {
		t.Fatalf("len(Blocks) = %d, want 10", len(grid.Blocks))
	}

	russia := grid.Blocks[7]
	if russia.Name != "TOTAL RUSSIA TIE LINES                      " {
		t.Errorf("Name = %q, want the name with its spaces", russia.Name)
	}

	want := gse.Entry{
		B1: "AllGeorg", B2: "Intercha", B3: "Java", Element: "P", Kind: "MvAvr03m",
		Value: "18.299999", Quality: "BLS",
	}
	if len(russia.Entries) != 4 || russia.Entries[2] != want {
		t.Errorf("Entries = %+v, want 4 with %+v third", russia.Entries, want)
	}
}

// GSEGRID-SC-002: The frequency file decodes as published.
func TestClientDecodesTheFrequency(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "frequency-16112026-1200.xml", http.StatusOK)

	frequency, err := client.Frequency(t.Context(), noon(t))
	if err != nil {
		t.Fatalf("Frequency() error = %v", err)
	}

	if len(frequency.Blocks) != 1 || len(frequency.Blocks[0].Entries) != 1 ||
		frequency.Blocks[0].Entries[0].Value != "49.990002" {
		t.Errorf("Blocks = %+v, want one entry 49.990002", frequency.Blocks)
	}
}

// GSEGRID-SC-003: A status that is not 200 is an *APIError, and a body that is not XML
// is an error.
func TestClientFailures(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		client, _ := serve(t, "not-found.html", status)

		_, err := client.Grid(t.Context(), noon(t))

		var apiErr *gse.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
			t.Errorf("Grid() error = %v, want an *APIError with %d", err, status)
		}

		if !errors.Is(err, gse.ErrUnexpectedStatus) {
			t.Errorf("errors.Is(%v, ErrUnexpectedStatus) = false", err)
		}
	}

	client, _ := serve(t, "", http.StatusOK)
	if _, err := client.Grid(t.Context(), noon(t)); err == nil {
		t.Error("Grid() with a body that is not XML gives no error")
	}
}
