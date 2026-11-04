package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	tbilisiID = 51
	kutaisiID = 52
	rustaviID = 55
	tsrt      = "TSRT"
	tbilisi   = "Tbilisi"
)

func clock() time.Time {
	return time.Date(2026, time.November, 5, 0, 28, 0, 0, kernel.TbilisiZone())
}

func code(tb testing.TB, text string) domain.StationCode {
	tb.Helper()

	parsed, err := domain.ParseStationCode(text)
	if err != nil {
		tb.Fatal(err)
	}

	return parsed
}

func hour(day, h int) time.Time {
	return time.Date(2026, time.November, day, h, 0, 0, 0, kernel.TbilisiZone())
}

// station builds a station with one PM10 reading of the level at the hour, or none.
func station(
	tb testing.TB,
	text string,
	city int,
	level domain.Level,
	at time.Time,
) domain.Station {
	tb.Helper()

	series := domain.Series{Pollutant: domain.PM10}
	if level != 0 {
		series.Readings = []domain.Reading{
			{Value: decimal.NewFromInt(int64(level) * 20), Time: at, Level: level},
		}
	}

	return domain.Station{
		Code: code(tb, text), CityID: city,
		Address: domain.Name{Georgian: text + " ქ.", English: text + " St."},
		Series:  []domain.Series{series},
	}
}

func network(tb testing.TB) app.Network {
	tb.Helper()

	recent := hour(5, 0)

	return app.Network{
		Stations: []domain.Station{
			station(tb, "KUTS", kutaisiID, domain.LevelModerate, recent),
			station(tb, tsrt, tbilisiID, domain.LevelPoor, hour(5, 5)),
			station(tb, "KZBG", tbilisiID, domain.LevelModerate, recent),
			station(tb, "VRKT", tbilisiID, domain.LevelFair, recent),
			station(tb, "RST18", rustaviID, 0, recent),
			station(tb, "AGMS", tbilisiID, domain.LevelPoor, hour(4, 19)),
			station(tb, "ORN01", tbilisiID, domain.LevelModerate, recent),
			station(tb, "ORN02", rustaviID, domain.LevelPoor, recent),
		},
		ReadAt: clock(),
	}
}

func cities() app.CityList {
	return app.CityList{
		Cities: []domain.City{
			{ID: 1, Name: domain.Name{Georgian: "თეთრი წყარო", English: "Tetritskaro"}},
			{ID: tbilisiID, Name: domain.Name{Georgian: "თბილისი", English: tbilisi}},
			{ID: kutaisiID, Name: domain.Name{Georgian: "ქუთაისი", English: "Kutaisi"}},
			{ID: rustaviID, Name: domain.Name{Georgian: "რუსთავი", English: "Rustavi"}},
		},
		ReadAt: clock(),
	}
}

// historyCall is one call of History, or of Recent with no hours.
type historyCall struct {
	Code     string
	From, To time.Time
}

type fakeReadings struct {
	current app.Network
	history app.Network
	err     error
	calls   []historyCall
	reads   int
}

func (f *fakeReadings) Current(context.Context) (app.Network, error) {
	f.reads++

	return f.current, f.err
}

func (f *fakeReadings) Recent(_ context.Context, code domain.StationCode) (app.Network, error) {
	f.calls = append(f.calls, historyCall{Code: code.String()})

	return f.history, f.err
}

func (f *fakeReadings) History(
	_ context.Context,
	code domain.StationCode,
	from, to time.Time,
) (app.Network, error) {
	f.calls = append(f.calls, historyCall{Code: code.String(), From: from, To: to})

	return f.history, f.err
}

type fakeCities struct {
	list app.CityList
	err  error
}

func (f fakeCities) Cities(context.Context) (app.CityList, error) {
	return f.list, f.err
}

func newGetQuality(t *testing.T) (*app.GetQuality, *fakeReadings) {
	t.Helper()

	readings := &fakeReadings{current: network(t)}

	return app.NewGetQuality(readings, fakeCities{list: cities()}, clock), readings
}

func codes(quality app.Quality) []string {
	got := make([]string, 0, len(quality.Stations))
	for _, item := range quality.Stations {
		got = append(got, item.Station.Code.String())
	}

	return got
}

// NEAAIR-SC-009: A city matches in English or Georgian, with no regard to case. Its
// stations come worst first, then by code.
func TestGetQualityOfACity(t *testing.T) {
	t.Parallel()

	for _, city := range []string{tbilisi, "tbilisi", " თბილისი "} {
		t.Run(city, func(t *testing.T) {
			t.Parallel()

			uses, _ := newGetQuality(t)

			quality, err := uses.Get(t.Context(), app.QualityQuery{City: city, Limit: 25})
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			want := []string{"AGMS", tsrt, "KZBG", "ORN01", "VRKT"}
			if diff := cmp.Diff(want, codes(quality)); diff != "" {
				t.Errorf("stations mismatch (-want +got):\n%s", diff)
			}

			first := quality.Stations[0]
			if first.City.Name.English != tbilisi || first.Level != domain.LevelPoor ||
				len(first.Readings) != 1 {
				t.Errorf("first station = %+v", first)
			}

			if quality.Total != 5 || !quality.ReadAt.Equal(clock()) {
				t.Errorf("Total = %d, ReadAt = %v", quality.Total, quality.ReadAt)
			}
		})
	}
}

// NEAAIR-SC-010: A city with no station gives the cities that have one, by English name.
func TestGetQualityRefusesAnUnknownCity(t *testing.T) {
	t.Parallel()

	for _, city := range []string{"Sochi", "Tetritskaro"} {
		uses, _ := newGetQuality(t)

		_, err := uses.Get(t.Context(), app.QualityQuery{City: city, Limit: 25})

		var cityError *app.CityError
		if !errors.As(err, &cityError) || !errors.Is(err, app.ErrUnknownCity) {
			t.Fatalf("Get(%q) error = %v, want a CityError", city, err)
		}

		names := make([]string, 0, len(cityError.Accepted))
		for _, accepted := range cityError.Accepted {
			names = append(names, accepted.Name.English)
		}

		if diff := cmp.Diff([]string{"Kutaisi", "Rustavi", tbilisi}, names); diff != "" {
			t.Errorf("accepted cities mismatch (-want +got):\n%s", diff)
		}
	}
}

// NEAAIR-SC-011: A station code gives its station. An unknown code, and a city with a
// station, are refused.
func TestGetQualityOfAStation(t *testing.T) {
	t.Parallel()

	uses, readings := newGetQuality(t)
	tsrtCode := code(t, tsrt)

	quality, err := uses.Get(t.Context(), app.QualityQuery{Station: &tsrtCode, Limit: 25})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if diff := cmp.Diff([]string{tsrt}, codes(quality)); diff != "" {
		t.Errorf("stations mismatch (-want +got):\n%s", diff)
	}

	unknown := code(t, "XXXX")
	if _, err := uses.Get(t.Context(), app.QualityQuery{Station: &unknown, Limit: 25}); !errors.Is(
		err, app.ErrUnknownStation,
	) {
		t.Errorf("Get(XXXX) error = %v, want ErrUnknownStation", err)
	}

	reads := readings.reads

	_, err = uses.Get(t.Context(), app.QualityQuery{City: tbilisi, Station: &tsrtCode, Limit: 25})
	if !errors.Is(err, app.ErrBothFilters) || readings.reads != reads {
		t.Errorf("Get(city and station) error = %v, reads %d, want ErrBothFilters and no read",
			err, readings.reads-reads)
	}
}

// NEAAIR-SC-012: No filter gives every station, worst first, with no level last. The limit
// cuts the list and keeps the total.
func TestGetQualityOfEveryStation(t *testing.T) {
	t.Parallel()

	uses, _ := newGetQuality(t)

	quality, err := uses.Get(t.Context(), app.QualityQuery{Limit: 25})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	want := []string{"ORN02", "AGMS", tsrt, "KUTS", "KZBG", "ORN01", "VRKT", "RST18"}
	if diff := cmp.Diff(want, codes(quality)); diff != "" {
		t.Errorf("stations mismatch (-want +got):\n%s", diff)
	}

	if quality.Total != 8 {
		t.Errorf("Total = %d, want 8", quality.Total)
	}

	cut, err := uses.Get(t.Context(), app.QualityQuery{Limit: 3})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(cut.Stations) != 3 || cut.Total != 8 {
		t.Errorf("limit 3: %d stations, Total %d, want 3 and 8", len(cut.Stations), cut.Total)
	}
}

// NEAAIR-SC-012: A stale source gives a stale answer. A failure passes through.
func TestGetQualityStaleAndFailure(t *testing.T) {
	t.Parallel()

	stale := network(t)
	stale.Stale = true

	uses := app.NewGetQuality(&fakeReadings{current: stale}, fakeCities{list: cities()}, clock)

	quality, err := uses.Get(t.Context(), app.QualityQuery{Limit: 25})
	if err != nil || !quality.Stale {
		t.Errorf("Get() = stale %t, %v, want a stale answer", quality.Stale, err)
	}

	failing := app.NewGetQuality(
		&fakeReadings{err: kernel.ErrSourceUnavailable}, fakeCities{list: cities()}, clock,
	)
	if _, err := failing.Get(t.Context(), app.QualityQuery{Limit: 25}); !errors.Is(
		err, kernel.ErrSourceUnavailable,
	) {
		t.Errorf("Get() error = %v, want ErrSourceUnavailable", err)
	}
}

func week(tb testing.TB) app.Network {
	tb.Helper()

	var pm10, pm25 []domain.Reading
	for at := hour(1, 0).Add(-24 * time.Hour); !at.After(hour(5, 6)); at = at.Add(time.Hour) {
		pm10 = append(pm10, domain.Reading{Value: decimal.NewFromInt(40), Time: at, Level: 3})
		pm25 = append(pm25, domain.Reading{Value: decimal.NewFromInt(15), Time: at, Level: 2})
	}

	tsrtStation := station(tb, tsrt, tbilisiID, 0, clock())
	tsrtStation.Series = []domain.Series{
		{Pollutant: domain.PM10, Readings: pm10},
		{Pollutant: domain.PM25, Readings: pm25},
	}

	return app.Network{Stations: []domain.Station{tsrtStation}, ReadAt: clock()}
}

func date(tb testing.TB, text string) *kernel.Date {
	tb.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		tb.Fatal(err)
	}

	return &parsed
}

func newListHistory(t *testing.T) (*app.ListHistory, *fakeReadings) {
	t.Helper()

	readings := &fakeReadings{history: week(t)}

	return app.NewListHistory(readings, fakeCities{list: cities()}, clock), readings
}

// checkSpan checks the count of readings and the time of the first and the last.
func checkSpan(t *testing.T, readings []domain.Reading, count int, first, last time.Time) {
	t.Helper()

	if len(readings) != count || !readings[0].Time.Equal(first) ||
		!readings[len(readings)-1].Time.Equal(last) {
		t.Errorf("%d readings from %v to %v, want %d from %v to %v", len(readings),
			readings[0].Time, readings[len(readings)-1].Time, count, first, last)
	}
}

// NEAAIR-SC-013: A period of dates reads from the first day at 00:00 to the last day at
// 23:00, and gives the last readings of each series, newest first.
func TestListHistoryOfDates(t *testing.T) {
	t.Parallel()

	uses, readings := newListHistory(t)

	history, err := uses.List(t.Context(), app.HistoryQuery{
		Station: code(t, tsrt), From: date(t, "2026-11-01"), To: date(t, "2026-11-04"), Limit: 24,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := []historyCall{{Code: tsrt, From: hour(1, 0), To: hour(4, 23)}}
	if diff := cmp.Diff(want, readings.calls); diff != "" {
		t.Errorf("calls mismatch (-want +got):\n%s", diff)
	}

	if len(history.Station.Series) != 2 {
		t.Fatalf("len(series) = %d, want 2", len(history.Station.Series))
	}

	checkSpan(t, history.Station.Series[0].Readings, 24, hour(4, 23), hour(4, 0))

	if history.Total[domain.PM10] != 96 || history.City.Name.English != tbilisi ||
		!history.From.Equal(hour(1, 0)) || !history.To.Equal(hour(4, 23)) {
		t.Errorf("Total = %v, City = %+v, period = %v to %v",
			history.Total, history.City, history.From, history.To)
	}
}

// NEAAIR-SC-014: No dates read the recent readings of the station, keep them from 24
// hours back, and end the period at the newest reading.
func TestListHistoryOfTheLastDay(t *testing.T) {
	t.Parallel()

	uses, readings := newListHistory(t)

	history, err := uses.List(t.Context(), app.HistoryQuery{Station: code(t, tsrt), Limit: 100})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if diff := cmp.Diff([]historyCall{{Code: tsrt}}, readings.calls); diff != "" {
		t.Errorf("calls mismatch (-want +got):\n%s", diff)
	}

	checkSpan(t, history.Station.Series[0].Readings, 31, hour(5, 6), hour(4, 0))

	if history.Total[domain.PM10] != 31 || !history.From.Equal(hour(4, 0)) ||
		!history.To.Equal(hour(5, 6)) {
		t.Errorf("Total of PM10 = %d, period = %v to %v, want 31 from %v to %v",
			history.Total[domain.PM10], history.From, history.To, hour(4, 0), hour(5, 6))
	}
}

// NEAAIR-SC-014: With no reading after the hour of the request, the period ends at
// that hour.
func TestListHistoryOfTheLastDayEndsAtTheRequest(t *testing.T) {
	t.Parallel()

	network := week(t)
	for index, series := range network.Stations[0].Series {
		network.Stations[0].Series[index].Readings = slices.DeleteFunc(series.Readings,
			func(r domain.Reading) bool { return r.Time.After(hour(4, 22)) })
	}

	uses := app.NewListHistory(&fakeReadings{history: network}, fakeCities{list: cities()}, clock)

	history, err := uses.List(t.Context(), app.HistoryQuery{Station: code(t, tsrt), Limit: 100})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if !history.To.Equal(hour(5, 0)) {
		t.Errorf("To = %v, want %v", history.To, hour(5, 0))
	}
}

// NEAAIR-SC-015: A bad period is refused before any call.
func TestListHistoryRefusesABadPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		from, to *kernel.Date
		message  string
	}{
		{
			name: "eight days", from: date(t, "2026-10-29"), to: date(t, "2026-11-05"),
			message: "from, to: 2026-10-29 to 2026-11-05 is 8 days. Give at most 7.",
		},
		{
			name: "reversed", from: date(t, "2026-11-03"), to: date(t, "2026-11-01"),
			message: "to: 2026-11-01 is before from 2026-11-03.",
		},
		{
			name: "too early", from: date(t, "2026-10-20"), to: date(t, "2026-10-22"),
			message: "from: 2026-10-20 is more than 7 days ago. Give 2026-10-29 or later.",
		},
		{
			name: "from alone", from: date(t, "2026-11-03"),
			message: "from, to: give both dates, or neither for the last 24 hours.",
		},
		{
			name: "to alone", to: date(t, "2026-11-03"),
			message: "from, to: give both dates, or neither for the last 24 hours.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uses, readings := newListHistory(t)

			_, err := uses.List(t.Context(), app.HistoryQuery{
				Station: code(t, tsrt), From: tt.from, To: tt.to, Limit: 24,
			})

			var periodError *app.PeriodError
			if !errors.As(err, &periodError) || !errors.Is(err, app.ErrInvalidPeriod) {
				t.Fatalf("List() error = %v, want a PeriodError", err)
			}

			if periodError.Message != tt.message || len(readings.calls) != 0 {
				t.Errorf("message = %q, calls = %d, want %q and none",
					periodError.Message, len(readings.calls), tt.message)
			}
		})
	}

	uses, readings := newListHistory(t)

	_, err := uses.List(t.Context(), app.HistoryQuery{
		Station: code(t, tsrt), From: date(t, "2026-10-29"), To: date(t, "2026-11-04"), Limit: 24,
	})
	if err != nil || len(readings.calls) != 1 {
		t.Errorf("List(7 days) error = %v, calls = %d, want one call", err, len(readings.calls))
	}
}

// NEAAIR-SC-016: A pollutant keeps its series only. An unknown station is refused.
func TestListHistoryOfAPollutant(t *testing.T) {
	t.Parallel()

	uses, _ := newListHistory(t)
	pm25 := domain.PM25

	history, err := uses.List(t.Context(), app.HistoryQuery{
		Station: code(t, tsrt), Pollutant: &pm25, Limit: 24,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(history.Station.Series) != 1 || history.Station.Series[0].Pollutant != domain.PM25 {
		t.Errorf("series = %+v, want PM2.5 only", history.Station.Series)
	}

	empty := app.NewListHistory(&fakeReadings{}, fakeCities{list: cities()}, clock)
	if _, err := empty.List(
		t.Context(),
		app.HistoryQuery{Station: code(t, "XXXX"), Limit: 24},
	); !errors.Is(
		err,
		app.ErrUnknownStation,
	) {
		t.Errorf("List(XXXX) error = %v, want ErrUnknownStation", err)
	}
}
