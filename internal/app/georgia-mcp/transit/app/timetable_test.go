package app_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// fixtureTimetable is the week of 2026-10-19 of the 385. Its first stop is also its
// last, with other times.
func fixtureTimetable() domain.Timetable {
	var days []domain.TimetableDay

	for date := mustDate("2026-10-19"); !date.After(mustDate("2026-10-25")); date = date.AddDays(1) {
		days = append(days, domain.TimetableDay{Date: date, Stops: []domain.StopTimes{
			{
				Stop:     tandzia,
				Position: 1,
				Times:    []domain.ClockTime{mustClock("6:45"), mustClock("6:55")},
			},
			{Stop: park, Position: 2, Times: []domain.ClockTime{mustClock("6:47")}},
			{Stop: tandzia, Position: 22, Times: []domain.ClockTime{mustClock("7:20")}},
		}})
	}

	return domain.Timetable{Days: days}
}

func clocks(times []domain.ClockTime) []string {
	texts := make([]string, 0, len(times))
	for _, clock := range times {
		texts = append(texts, clock.String())
	}

	return texts
}

func newGetTimetable(now string) (*app.GetTimetable, *fakeRoutes) {
	routes := &fakeRoutes{directions: metroDirections(), timetable: fixtureTimetable()}

	return app.NewGetTimetable(newNetwork(), routes, fixedNow(now)), routes
}

// TTCTRANS-SC-030: The times of a stop come from its first position. No date uses the
// date in Tbilisi.
func TestGetTimetable(t *testing.T) {
	t.Parallel()

	use, _ := newGetTimetable("2026-10-20 08:00:00")

	got, err := use.Get(t.Context(), app.TimetableQuery{
		Route: "385", Stop: tandzia, Date: mustDate("2026-10-19"),
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if diff := cmp.Diff([]string{"06:45", "06:55"}, clocks(got.Times)); diff != "" {
		t.Errorf("times mismatch (-want +got):\n%s", diff)
	}

	today, err := use.Get(t.Context(), app.TimetableQuery{Route: "385", Stop: "1001"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if today.Date != mustDate("2026-10-20") || today.Stop.ID != tandzia || today.Direction.ID != 1 {
		t.Errorf("Get() = %+v, want 2026-10-20 at 1:1001 in direction 1", today)
	}
}

// TTCTRANS-SC-031: A route of two directions needs one. A route of one direction uses
// it. A stop must be on the direction.
func TestGetTimetableChecksTheDirectionAndTheStop(t *testing.T) {
	t.Parallel()

	use, routes := newGetTimetable("2026-10-20 08:00:00")

	_, err := use.Get(
		t.Context(),
		app.TimetableQuery{Route: "1", Kind: domain.KindMetro, Stop: tandzia},
	)

	direction, ok := errors.AsType[*app.DirectionError](err)
	if !ok || !errors.Is(err, app.ErrUnknownDirection) || len(direction.Directions) != 2 {
		t.Errorf("Get() error = %v, want *DirectionError with 2 directions", err)
	}

	unknown := 5
	if _, err := use.Get(t.Context(), app.TimetableQuery{
		Route: "385", Direction: &unknown, Stop: tandzia,
	}); !errors.Is(err, app.ErrUnknownDirection) {
		t.Errorf("Get() error = %v, want ErrUnknownDirection", err)
	}

	if _, err := use.Get(t.Context(), app.TimetableQuery{Route: "385", Stop: "1:1055"}); !errors.Is(
		err, app.ErrStopNotOnDirection,
	) {
		t.Errorf("Get() error = %v, want ErrStopNotOnDirection", err)
	}

	if routes.asked != 1 {
		t.Errorf("the timetable of direction %d was asked, want 1", routes.asked)
	}
}

// TTCTRANS-SC-032: A date outside the published week is refused with the first and
// the last date of the week.
func TestGetTimetableRefusesADateOutsideTheWeek(t *testing.T) {
	t.Parallel()

	use, _ := newGetTimetable("2026-10-20 08:00:00")

	if _, err := use.Get(t.Context(), app.TimetableQuery{
		Route: "385", Stop: tandzia, Date: mustDate("2026-10-25"),
	}); err != nil {
		t.Errorf("Get(2026-10-25) error = %v", err)
	}

	_, err := use.Get(t.Context(), app.TimetableQuery{
		Route: "385", Stop: tandzia, Date: mustDate("2026-10-26"),
	})

	outside, ok := errors.AsType[*app.DateOutsideTimetableError](err)
	if !ok || !errors.Is(err, app.ErrDateOutsideTimetable) {
		t.Fatalf("Get(2026-10-26) error = %v, want *DateOutsideTimetableError", err)
	}

	want := app.DateOutsideTimetableError{
		Date: mustDate("2026-10-26"), First: mustDate("2026-10-19"), Last: mustDate("2026-10-25"),
	}
	if diff := cmp.Diff(want, *outside, cmp.AllowUnexported(kernel.Date{})); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}
