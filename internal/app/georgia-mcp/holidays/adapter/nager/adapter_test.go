package nager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	nageradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/adapter/nager"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	nagerclient "github.com/abgeo/georgia-mcp/pkg/nager"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const testdata = "../../../../../../pkg/nager/testdata/"

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC)
}

// serve answers each request with the file and the status.
func serve(t *testing.T, file string, status int) *nageradapter.Adapter {
	t.Helper()

	body, err := os.ReadFile(testdata + file) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return nageradapter.NewAdapter(nagerclient.New(vendorhttp.WithBaseURL(server.URL)), clock)
}

// HOLIDAY-SC-006: The recorded answer of 2026 gives 17 holidays in date order, read in
// Tbilisi. An item that is not a public holiday is left out.
func TestHolidays(t *testing.T) {
	t.Parallel()

	for _, file := range []string{"ge-2026.json", "ge-observance.json"} {
		t.Run(file, func(t *testing.T) {
			t.Parallel()

			calendar, err := serve(t, file, http.StatusOK).Holidays(t.Context(), 2026)
			if err != nil {
				t.Fatalf("Holidays() error = %v", err)
			}

			if calendar.Year != 2026 || len(calendar.Holidays) != 17 {
				t.Fatalf(
					"Holidays() = %d holidays of %d, want 17 of 2026",
					len(calendar.Holidays),
					calendar.Year,
				)
			}

			checkCalendar(t, calendar)
		})
	}
}

func checkCalendar(t *testing.T, calendar app.Calendar) {
	t.Helper()

	first := calendar.Holidays[0]
	if first.Date.String() != "2026-01-01" ||
		first.Name.In(kernel.LanguageGeorgian) != "ახალი წელი" ||
		first.Name.In(kernel.LanguageEnglish) != "New Year's Day" {
		t.Errorf(
			"Holidays()[0] = %s %q, want 2026-01-01 New Year's Day",
			first.Date,
			first.Name.In(kernel.LanguageEnglish),
		)
	}

	for i := 1; i < len(calendar.Holidays); i++ {
		if calendar.Holidays[i].Date.Before(calendar.Holidays[i-1].Date) {
			t.Errorf("holiday %d is before holiday %d", i, i-1)
		}
	}

	if name, offset := calendar.ReadAt.Zone(); name != kernel.TbilisiZone().String() ||
		offset != 4*60*60 {
		t.Errorf("ReadAt zone = %s %d, want Tbilisi", name, offset)
	}

	if !calendar.ReadAt.Equal(clock()) {
		t.Errorf("ReadAt = %s, want %s", calendar.ReadAt, clock())
	}
}

// HOLIDAY-SC-006: An empty answer, a broken date, a date of another year, and a client
// error are each a failure of the source.
func TestHolidaysFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file   string
		status int
	}{
		{file: "ge-empty.json", status: http.StatusOK},
		{file: "ge-bad-date.json", status: http.StatusOK},
		{file: "ge-other-year.json", status: http.StatusOK},
		{file: "year-400.json", status: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			t.Parallel()

			_, err := serve(t, test.file, test.status).Holidays(t.Context(), 2026)
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("Holidays() error = %v, want ErrSourceUnavailable", err)
			}
		})
	}
}
