package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	newYear   = "New Year's Day"
	today     = "2026-11-15"
	newYear27 = "2027-01-01"
	george    = "2026-11-23"
)

type holidayDay struct {
	monthDay string
	name     string
}

// days gives the month and the day of each holiday of 2026, by English name. The fake
// gives the same days in each year.
func days() []holidayDay {
	return []holidayDay{
		{"01-01", newYear},
		{"01-02", newYear},
		{"01-07", "Christmas Day (Orthodox)"},
		{"01-19", "Epiphany (Orthodox)"},
		{"03-03", "Mother's Day"},
		{"03-08", "International Women's Day"},
		{"04-09", "National Unity Day"},
		{"04-10", "Good Friday"},
		{"04-11", "Holy Saturday"},
		{"04-12", "Easter Sunday"},
		{"04-13", "Easter Monday"},
		{"05-09", "Day of Victory over Fascism"},
		{"05-12", "Saint Andrew the First-Called Day"},
		{"05-26", "Independence Day"},
		{"08-28", "Saint Mary's Day"},
		{"10-14", "Day of Svetitskhoveli Cathedra"},
		{"11-23", "Saint George's Day"},
	}
}

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

func date(tb testing.TB, text string) kernel.Date {
	tb.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		tb.Fatal(err)
	}

	return parsed
}

func calendar(tb testing.TB, year int) app.Calendar {
	tb.Helper()

	holidays := make([]domain.Holiday, 0, len(days()))

	for _, day := range days() {
		name, err := domain.NewHolidayName("", day.name)
		if err != nil {
			tb.Fatal(err)
		}

		holidays = append(holidays, domain.Holiday{
			Date: date(tb, fmt.Sprintf("%d-%s", year, day.monthDay)),
			Name: name,
		})
	}

	return app.Calendar{Year: year, Holidays: holidays, ReadAt: clock()}
}

type fakeSource struct {
	tb    testing.TB
	reads []int
	read  map[int]time.Time
	stale map[int]bool
	err   error
}

func (f *fakeSource) Holidays(_ context.Context, year int) (app.Calendar, error) {
	f.reads = append(f.reads, year)

	if f.err != nil {
		return app.Calendar{}, f.err
	}

	result := calendar(f.tb, year)
	if at, ok := f.read[year]; ok {
		result.ReadAt = at
	}

	result.Stale = f.stale[year]

	return result, nil
}

type entry struct {
	Date string
	Name string
}

func entries(holidays []domain.Holiday) []entry {
	result := make([]entry, 0, len(holidays))
	for _, holiday := range holidays {
		result = append(
			result,
			entry{Date: holiday.Date.String(), Name: holiday.Name.In(kernel.LanguageEnglish)},
		)
	}

	return result
}

// HOLIDAY-SC-007: The supported years run from 2025 to the year after the date in Tbilisi.
func TestSupportedYears(t *testing.T) {
	t.Parallel()

	tests := []struct {
		today     string
		wantFirst int
		wantLast  int
	}{
		{today: today, wantFirst: 2025, wantLast: 2027},
		{today: "2026-12-31", wantFirst: 2025, wantLast: 2027},
		{today: newYear27, wantFirst: 2025, wantLast: 2028},
	}

	for _, test := range tests {
		t.Run(test.today, func(t *testing.T) {
			t.Parallel()

			first, last := app.SupportedYears(date(t, test.today))
			if first != test.wantFirst || last != test.wantLast {
				t.Errorf(
					"SupportedYears() = %d, %d, want %d, %d",
					first,
					last,
					test.wantFirst,
					test.wantLast,
				)
			}
		})
	}

	evening := time.Date(2026, time.December, 31, 21, 0, 0, 0, time.UTC)
	if got := kernel.Today(evening).String(); got != newYear27 {
		t.Errorf("Today(2026-12-31T21:00:00Z) = %s, want 2027-01-01", got)
	}
}

// HOLIDAY-SC-008: A year gives its 17 holidays. No input gives the current year.
func TestListYear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query app.ListQuery
	}{
		{name: "2026", query: app.ListQuery{Year: new(2026)}},
		{name: "no input", query: app.ListQuery{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			source := &fakeSource{tb: t}

			list, err := app.NewListHolidays(source, clock).List(t.Context(), test.query)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			got := entries(list.Holidays)
			if len(got) != 17 || got[0].Date != "2026-01-01" || got[16].Date != george {
				t.Errorf("List() = %v, want 17 holidays from 2026-01-01 to 2026-11-23", got)
			}

			if diff := cmp.Diff([]int{2026}, source.reads); diff != "" {
				t.Errorf("reads mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// HOLIDAY-SC-009: A year or a date outside the supported years, and a year with a from
// date, are refused before any read.
func TestListRefuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    app.ListQuery
		wantErr  error
		wantYear int
	}{
		{
			name:     "2024",
			query:    app.ListQuery{Year: new(2024)},
			wantErr:  app.ErrUnsupportedYear,
			wantYear: 2024,
		},
		{
			name:     "2028",
			query:    app.ListQuery{Year: new(2028)},
			wantErr:  app.ErrUnsupportedYear,
			wantYear: 2028,
		},
		{
			name: "from 2028", query: app.ListQuery{From: new(date(t, "2028-01-01")), Limit: 1},
			wantErr: app.ErrUnsupportedYear, wantYear: 2028,
		},
		{
			name: "both", query: app.ListQuery{Year: new(2026), From: new(date(t, today))},
			wantErr: app.ErrBothFilters,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			source := &fakeSource{tb: t}

			_, err := app.NewListHolidays(source, clock).List(t.Context(), test.query)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("List() error = %v, want %v", err, test.wantErr)
			}

			if test.wantYear != 0 {
				var yearErr *app.YearError

				want := &app.YearError{Year: test.wantYear, First: 2025, Last: 2027}
				if !errors.As(err, &yearErr) || *yearErr != *want {
					t.Errorf("List() error = %#v, want %#v", err, want)
				}
			}

			if len(source.reads) != 0 {
				t.Errorf("reads = %v, want none", source.reads)
			}
		})
	}
}

// HOLIDAY-SC-009: Get refuses a date outside the supported years.
func TestGetRefuses(t *testing.T) {
	t.Parallel()

	source := &fakeSource{tb: t}

	_, err := app.NewGetDay(source, clock).
		Get(t.Context(), app.DayQuery{Date: new(date(t, "2028-01-01"))})

	var yearErr *app.YearError
	if !errors.As(err, &yearErr) ||
		*yearErr != (app.YearError{Year: 2028, First: 2025, Last: 2027}) {
		t.Fatalf("Get() error = %v, want a *YearError for 2028", err)
	}

	if len(source.reads) != 0 {
		t.Errorf("reads = %v, want none", source.reads)
	}
}

type nextCase struct {
	name      string
	from      string
	limit     int
	want      []string
	wantReads []int
}

func nextCases() []nextCase {
	return []nextCase{
		{
			name:      "next",
			from:      today,
			limit:     1,
			want:      []string{george},
			wantReads: []int{2026},
		},
		{
			name:      "on the day",
			from:      george,
			limit:     1,
			want:      []string{george},
			wantReads: []int{2026},
		},
		{
			name:      "new year",
			from:      "2026-12-31",
			limit:     1,
			want:      []string{newYear27},
			wantReads: []int{2026, 2027},
		},
		{
			name: "cross", from: today, limit: 3,
			want: []string{george, newYear27, "2027-01-02"}, wantReads: []int{2026, 2027},
		},
		{
			name:      "last year",
			from:      "2027-11-15",
			limit:     20,
			want:      []string{"2027-11-23"},
			wantReads: []int{2027},
		},
	}
}

// HOLIDAY-SC-010, HOLIDAY-SC-011: The next holidays start on the from date, cross into
// the next year, and stop at the end of the last supported year.
func TestListNext(t *testing.T) {
	t.Parallel()

	tests := nextCases()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			source := &fakeSource{tb: t}

			list, err := app.NewListHolidays(source, clock).List(
				t.Context(), app.ListQuery{From: new(date(t, test.from)), Limit: test.limit},
			)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			got := make([]string, 0, len(list.Holidays))
			for _, holiday := range list.Holidays {
				got = append(got, holiday.Date.String())
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("dates mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(test.wantReads, source.reads); diff != "" {
				t.Errorf("reads mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// HOLIDAY-SC-011: A list of two calendars carries the older read and the stale flag of
// either.
func TestListJoinsTheReads(t *testing.T) {
	t.Parallel()

	older := clock().Add(-48 * time.Hour)
	source := &fakeSource{
		tb:    t,
		read:  map[int]time.Time{2027: older},
		stale: map[int]bool{2027: true},
	}

	list, err := app.NewListHolidays(source, clock).List(
		t.Context(), app.ListQuery{From: new(date(t, today)), Limit: 3},
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if !list.ReadAt.Equal(older) || !list.Stale {
		t.Errorf("ReadAt, Stale = %s, %t, want %s, true", list.ReadAt, list.Stale, older)
	}
}

// HOLIDAY-SC-012: A date gives its holiday, or none. No date asks about today.
func TestGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		date     *kernel.Date
		wantDate string
		wantName string
	}{
		{
			name:     "independence",
			date:     new(date(t, "2026-05-26")),
			wantDate: "2026-05-26",
			wantName: "Independence Day",
		},
		{
			name:     "new year",
			date:     new(date(t, "2026-01-02")),
			wantDate: "2026-01-02",
			wantName: newYear,
		},
		{name: "no holiday", date: new(date(t, "2026-05-27")), wantDate: "2026-05-27"},
		{name: "today", wantDate: today},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			day, err := app.NewGetDay(&fakeSource{tb: t}, clock).
				Get(t.Context(), app.DayQuery{Date: test.date})
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if day.Date.String() != test.wantDate {
				t.Errorf("Date = %s, want %s", day.Date, test.wantDate)
			}

			name := ""
			if day.Holiday != nil {
				name = day.Holiday.Name.In(kernel.LanguageEnglish)
			}

			if name != test.wantName {
				t.Errorf("holiday = %q, want %q", name, test.wantName)
			}
		})
	}
}

// HOLIDAY-SC-012: A failure of the source passes through.
func TestSourceFails(t *testing.T) {
	t.Parallel()

	source := &fakeSource{tb: t, err: kernel.ErrSourceUnavailable}

	if _, err := app.NewListHolidays(source, clock).
		List(t.Context(), app.ListQuery{}); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("List() error = %v, want ErrSourceUnavailable", err)
	}

	if _, err := app.NewGetDay(source, clock).
		Get(t.Context(), app.DayQuery{}); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("Get() error = %v, want ErrSourceUnavailable", err)
	}
}
