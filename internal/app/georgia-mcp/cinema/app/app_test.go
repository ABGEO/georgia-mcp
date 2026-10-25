package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	galleriaID     = "5a1583c9956167428b722f81"
	eastPointID    = "55e0d0da956167a5a79cc43c"
	cityMallID     = "5d8fb2ea95616708ef1cd194"
	streetFighter  = "6ab27d3fde5280679e341878"
	duneID         = "6adc8b3cde5280bf425b94a7"
	galleria2215   = "cag_5/27-10-2026/22:15"
	closedCinemaID = "54a19c820bc7cb374e6436c4"
)

func at(t *testing.T, hour int) time.Time {
	t.Helper()

	return time.Date(2026, time.October, 27, hour, 0, 0, 0, kernel.TbilisiZone())
}

func mustCinemaCode(t *testing.T, text string) domain.CinemaCode {
	t.Helper()

	code, err := domain.ParseCinemaCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return code
}

func mustMovieCode(t *testing.T, text string) domain.MovieCode {
	t.Helper()

	code, err := domain.ParseMovieCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return code
}

func mustSessionCode(t *testing.T, text string) domain.SessionCode {
	t.Helper()

	code, err := domain.ParseSessionCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return code
}

func mustPrice(t *testing.T, text string) domain.Price {
	t.Helper()

	price, err := domain.NewPrice(text)
	if err != nil {
		t.Fatal(err)
	}

	return price
}

// fakeCinemas answers three open cinemas.
type fakeCinemas struct{ t *testing.T }

func (f fakeCinemas) Cinemas(context.Context) (app.CinemaList, error) {
	return app.CinemaList{Cinemas: []domain.Cinema{
		{
			ID:   galleriaID,
			Code: mustCinemaCode(f.t, "galleria"),
			Name: domain.Text{English: "Cavea Galleria"},
		},
		{
			ID:   eastPointID,
			Code: mustCinemaCode(f.t, "east_point"),
			Name: domain.Text{English: "Cavea East Point"},
		},
		{
			ID:   cityMallID,
			Code: mustCinemaCode(f.t, "city_mall_saburtalo"),
			Name: domain.Text{English: "Cavea City Mall Saburtalo"},
		},
	}, ReadAt: at(f.t, 12)}, nil
}

// fakeMovies answers Street Fighter now and Dune soon, and a list of count movies for a
// search. It records each search text.
type fakeMovies struct {
	t      *testing.T
	count  int
	texts  []string
	status []domain.MovieStatus
}

func (f *fakeMovies) Movies(_ context.Context, status domain.MovieStatus) (app.MovieList, error) {
	f.status = append(f.status, status)

	if f.count > 0 {
		return app.MovieList{Movies: numberedMovies(f.t, f.count)}, nil
	}

	if status == domain.ComingSoon {
		return app.MovieList{Movies: []domain.Movie{
			{ID: duneID, Code: mustMovieCode(f.t, "dune-part-three"), Status: domain.ComingSoon},
		}}, nil
	}

	return app.MovieList{Movies: []domain.Movie{
		{
			ID:     streetFighter,
			Code:   mustMovieCode(f.t, "street-fighter"),
			Status: domain.NowShowing,
			Name:   domain.Text{English: "Street Fighter"},
		},
	}}, nil
}

func (f *fakeMovies) Search(_ context.Context, text string) (app.MovieList, error) {
	f.texts = append(f.texts, text)

	return app.MovieList{Movies: numberedMovies(f.t, f.count)}, nil
}

func numberedMovies(t *testing.T, count int) []domain.Movie {
	t.Helper()

	movies := make([]domain.Movie, count)
	for index := range movies {
		movies[index] = domain.Movie{Code: mustMovieCode(t, fmt.Sprintf("movie-%d", index))}
	}

	return movies
}

// fakeSessions answers the sessions to both methods, or priced to MovieSessions when it
// is set, and counts each call.
type fakeSessions struct {
	sessions []domain.Session
	// priced is the answer of MovieSessions when it is set.
	priced   []domain.Session
	stale    bool
	calls    int
	movieIDs []string
}

func (f *fakeSessions) Sessions(context.Context) (app.SessionList, error) {
	f.calls++

	return app.SessionList{Sessions: f.sessions, Stale: f.stale}, nil
}

func (f *fakeSessions) MovieSessions(_ context.Context, movieID string) (app.SessionList, error) {
	f.calls++
	f.movieIDs = append(f.movieIDs, movieID)

	if f.priced != nil {
		return app.SessionList{Sessions: f.priced, Stale: f.stale}, nil
	}

	return app.SessionList{Sessions: f.sessions, Stale: f.stale}, nil
}

// fakeSeats answers a level for each code and a fixed seat map, and counts each call.
type fakeSeats struct {
	seats     []domain.Seat
	failing   bool
	occupancy [][]domain.SessionCode
	maps      int
}

func (f *fakeSeats) Occupancy(
	_ context.Context,
	codes []domain.SessionCode,
) (map[domain.SessionCode]domain.OccupancyLevel, time.Time, error) {
	f.occupancy = append(f.occupancy, codes)

	if f.failing {
		return nil, time.Time{}, fmt.Errorf("calling Cavea: %w", kernel.ErrSourceUnavailable)
	}

	levels := map[domain.SessionCode]domain.OccupancyLevel{}
	for _, code := range codes {
		levels[code] = domain.FillingUp
	}

	return levels, time.Time{}, nil
}

func (f *fakeSeats) SeatMap(context.Context, domain.SessionCode) ([]domain.Seat, time.Time, error) {
	f.maps++

	return f.seats, time.Time{}, nil
}

func session(
	t *testing.T,
	code, movieID, cinemaID string,
	audio domain.AudioLanguage,
) domain.Session {
	t.Helper()

	return domain.Session{
		Code:     mustSessionCode(t, code),
		MovieID:  movieID,
		CinemaID: cinemaID,
		Audio:    audio,
		Prices:   map[domain.SeatClass]domain.Price{domain.Standard: mustPrice(t, "20")},
	}
}

func streetFighterSessions(t *testing.T) []domain.Session {
	t.Helper()

	return []domain.Session{
		session(t, "cep_1/27-10-2026/22:00", streetFighter, eastPointID, domain.AudioEnglish),
		session(t, "cag_3/27-10-2026/21:45", streetFighter, galleriaID, domain.AudioEnglish),
		session(t, "cag_5/27-10-2026/16:45", streetFighter, galleriaID, domain.AudioEnglish),
		session(t, galleria2215, streetFighter, galleriaID, domain.AudioGeorgian),
		session(t, "ccm_x/27-10-2026/21:45", streetFighter, cityMallID, domain.AudioEnglish),
		session(t, "ccm_x/26-10-2026/21:45", streetFighter, cityMallID, domain.AudioEnglish),
		session(t, "rus_1/27-10-2026/21:00", streetFighter, closedCinemaID, domain.AudioEnglish),
		session(t, "cep_2/27-10-2026/21:30", "unknown-movie", eastPointID, domain.AudioEnglish),
	}
}

func codes(sessions []app.ScheduledSession) []string {
	texts := make([]string, 0, len(sessions))
	for _, item := range sessions {
		texts = append(texts, item.Session.Code.String())
	}

	return texts
}

// CAVCINE-SC-015: The sessions of a movie on a date start after now, match the filters,
// and come in the order of the start, then the cinema code.
func TestListSessionsKeepsTheLaterSessionsInOrder(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessions{sessions: streetFighterSessions(t)}
	seats := &fakeSeats{}
	movie := mustMovieCode(t, "street-fighter")
	audio := domain.AudioEnglish

	list := app.NewListSessions(fakeCinemas{t}, &fakeMovies{t: t}, sessions, seats,
		func() time.Time { return at(t, 20) }, slog.New(slog.DiscardHandler))

	date, _ := kernel.ParseDate("2026-10-27")

	got, err := list.List(
		t.Context(),
		app.SessionQuery{Movie: &movie, Date: date, Audio: &audio, Limit: 30},
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := []string{"ccm_x/27-10-2026/21:45", "cag_3/27-10-2026/21:45", "cep_1/27-10-2026/22:00"}
	if !reflect.DeepEqual(codes(got.Sessions), want) || got.Total != 3 {
		t.Errorf("List() = %v with total %d, want %v", codes(got.Sessions), got.Total, want)
	}

	if !reflect.DeepEqual(sessions.movieIDs, []string{streetFighter}) {
		t.Errorf("MovieSessions() IDs = %v, want the ID of Street Fighter", sessions.movieIDs)
	}

	first := got.Sessions[0]
	if first.Cinema.Code.String() != "city_mall_saburtalo" ||
		first.Movie.Code.String() != "street-fighter" {
		t.Errorf("the first session = %+v, want City Mall and Street Fighter", first)
	}

	yesterday, _ := kernel.ParseDate("2026-10-26")
	if _, err := list.List(
		t.Context(),
		app.SessionQuery{Date: yesterday, Limit: 30},
	); !errors.Is(
		err,
		app.ErrPastDate,
	) {
		t.Errorf("List(2026-10-26) error = %v, want ErrPastDate", err)
	}
}

// CAVCINE-SC-015: A list with no movie reads the sessions of every movie, cuts the list
// to the limit, and counts every match.
func TestListSessionsWithNoMovieReadsEveryMovie(t *testing.T) {
	t.Parallel()

	sessions := &fakeSessions{sessions: streetFighterSessions(t), stale: true}
	list := app.NewListSessions(fakeCinemas{t}, &fakeMovies{t: t}, sessions, &fakeSeats{},
		func() time.Time { return at(t, 12) }, slog.New(slog.DiscardHandler))

	got, err := list.List(t.Context(), app.SessionQuery{Limit: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := []string{"cag_5/27-10-2026/16:45", "ccm_x/27-10-2026/21:45"}
	if !reflect.DeepEqual(codes(got.Sessions), want) || got.Total != 5 || !got.Stale {
		t.Errorf("List() = %v, total %d, stale %t, want %v, 5, true",
			codes(got.Sessions), got.Total, got.Stale, want)
	}

	if len(sessions.movieIDs) != 0 {
		t.Errorf("MovieSessions() calls = %v, want none", sessions.movieIDs)
	}
}

// CAVCINE-SC-016: An unknown movie or cinema is refused before the read of the sessions.
func TestListSessionsRefusesAnUnknownCode(t *testing.T) {
	t.Parallel()

	movie := mustMovieCode(t, "no-such-movie")
	cinema := mustCinemaCode(t, "no_such")

	tests := []struct {
		name  string
		query app.SessionQuery
		want  error
	}{
		{name: "movie", query: app.SessionQuery{Movie: &movie}, want: app.ErrUnknownMovie},
		{name: "cinema", query: app.SessionQuery{Cinema: &cinema}, want: app.ErrUnknownCinema},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sessions := &fakeSessions{}
			list := app.NewListSessions(fakeCinemas{t}, &fakeMovies{t: t}, sessions, &fakeSeats{},
				func() time.Time { return at(t, 12) }, slog.New(slog.DiscardHandler))

			if _, err := list.List(
				t.Context(),
				tt.query,
			); !errors.Is(err, tt.want) ||
				sessions.calls != 0 {
				t.Errorf(
					"List() error = %v with %d calls, want %v and no call",
					err,
					sessions.calls,
					tt.want,
				)
			}
		})
	}
}

// warning is a record at warn level of a JSON log.
type warning struct {
	Message  string
	Sessions float64
	HasError bool
}

func warnings(t *testing.T, sink *bytes.Buffer) []warning {
	t.Helper()

	found := []warning{}

	for line := range bytes.Lines(sink.Bytes()) {
		var record struct {
			Level    string  `json:"level"`
			Message  string  `json:"msg"`
			Sessions float64 `json:"sessions"`
			Error    any     `json:"error"`
		}
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatal(err)
		}

		if record.Level == slog.LevelWarn.String() {
			found = append(found, warning{record.Message, record.Sessions, record.Error != nil})
		}
	}

	return found
}

// CAVCINE-SC-017: Each kept session carries its level. When the levels fail, the
// sessions stay with no level, and the log holds one warning with the error.
func TestListSessionsAddsTheOccupancy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		failing   bool
		occupancy bool
		level     domain.OccupancyLevel
		warnings  []warning
	}{
		{name: "answer", occupancy: true, level: domain.FillingUp, warnings: []warning{}},
		{
			name: "failure", failing: true,
			warnings: []warning{{"Cavea did not give the occupancy levels", 5, true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seats := &fakeSeats{failing: tt.failing}
			sink := &bytes.Buffer{}
			list := app.NewListSessions(fakeCinemas{t}, &fakeMovies{t: t},
				&fakeSessions{sessions: streetFighterSessions(t)}, seats,
				func() time.Time { return at(t, 12) }, slog.New(slog.NewJSONHandler(sink, nil)))

			got, err := list.List(t.Context(), app.SessionQuery{Limit: 30})
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if len(seats.occupancy) != 1 || len(seats.occupancy[0]) != 5 {
				t.Fatalf(
					"Occupancy() calls = %v, want one call with the 5 kept codes",
					seats.occupancy,
				)
			}

			if got.Occupancy != tt.occupancy || got.Sessions[0].Occupancy != tt.level {
				t.Errorf("Occupancy = %t, the first level %q, want %t and %q",
					got.Occupancy, got.Sessions[0].Occupancy, tt.occupancy, tt.level)
			}

			if records := warnings(t, sink); !reflect.DeepEqual(records, tt.warnings) {
				t.Errorf("warnings = %+v, want %+v", records, tt.warnings)
			}
		})
	}
}

// hall returns the seats of Galleria hall 5: 41 Standard with 3 taken, 34 Status with 9
// taken, and 16 VIP, all taken.
func hall() []domain.Seat {
	seats := []domain.Seat{}

	add := func(class domain.SeatClass, count, taken int) {
		for number := range count {
			seats = append(
				seats,
				domain.Seat{Row: "1", Number: number + 1, Class: class, Taken: number < taken},
			)
		}
	}

	add(domain.VIP, 16, 16)
	add(domain.Standard, 41, 3)
	add(domain.Status, 34, 9)

	return seats
}

// CAVCINE-SC-018: The seat map counts the free seats of each class, and gives the prices
// of the session of the movie.
func TestGetSeatMapCountsTheFreeSeats(t *testing.T) {
	t.Parallel()

	page := session(t, galleria2215, streetFighter, galleriaID, domain.AudioGeorgian)
	priced := page
	priced.Prices = map[domain.SeatClass]domain.Price{
		domain.Economy: mustPrice(t, "20"), domain.Standard: mustPrice(t, "20"),
		domain.VIP: mustPrice(t, "29"), domain.Status: mustPrice(t, "22"),
	}

	sessions := &fakeSessions{sessions: []domain.Session{page}, priced: []domain.Session{priced}}
	seats := &fakeSeats{seats: hall()}
	get := app.NewGetSeatMap(fakeCinemas{t}, &fakeMovies{t: t}, sessions, seats,
		func() time.Time { return at(t, 20) })

	got, err := get.Get(t.Context(), mustSessionCode(t, galleria2215))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	type count struct {
		Class       domain.SeatClass
		Price       string
		Seats, Free int
	}

	counts := make([]count, 0, len(got.Classes))
	for _, class := range got.Classes {
		counts = append(counts, count{class.Class, class.Price.String(), class.Seats, class.Free})
	}

	want := []count{
		{domain.Standard, "20", 41, 38},
		{domain.VIP, "29", 16, 0},
		{domain.Status, "22", 34, 25},
	}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("Classes = %+v, want %+v", counts, want)
	}

	if len(got.Seats) != 91 || got.Session.Cinema.Code.String() != "galleria" ||
		!reflect.DeepEqual(sessions.movieIDs, []string{streetFighter}) {
		t.Errorf(
			"Get() = %d seats in %+v, movie reads %v",
			len(got.Seats),
			got.Session.Cinema,
			sessions.movieIDs,
		)
	}
}

// CAVCINE-SC-018: A session that started, or that Cavea does not list, is refused with no
// call of the seats.
func TestGetSeatMapRefusesWithNoCallOfTheSeats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
		want error
	}{
		{name: "started", code: "cag_5/27-10-2026/16:45", want: app.ErrSessionStarted},
		{name: "unknown", code: "cag_5/28-10-2026/22:15", want: app.ErrUnknownSession},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seats := &fakeSeats{seats: hall()}
			get := app.NewGetSeatMap(fakeCinemas{t}, &fakeMovies{t: t},
				&fakeSessions{sessions: streetFighterSessions(t)}, seats,
				func() time.Time { return at(t, 20) })

			if _, err := get.Get(
				t.Context(),
				mustSessionCode(t, tt.code),
			); !errors.Is(err, tt.want) ||
				seats.maps != 0 {
				t.Errorf(
					"Get() error = %v with %d seat calls, want %v and none",
					err,
					seats.maps,
					tt.want,
				)
			}
		})
	}
}

// CAVCINE-SC-026: A search text has 2 to 100 runes after the spaces at its ends go, and a
// list stops at the limit with the count of every match.
func TestSearchAndListMovies(t *testing.T) {
	t.Parallel()

	movies := &fakeMovies{t: t, count: 25}
	search := app.NewSearchMovies(movies, beforeDocumentaries)

	for _, text := range []string{"a", "  a  ", strings.Repeat("ა", 101)} {
		if _, err := search.Search(
			t.Context(),
			app.SearchQuery{Text: text, Limit: 20},
		); !errors.Is(
			err,
			app.ErrInvalidQuery,
		) {
			t.Errorf("Search(%q) error = %v, want ErrInvalidQuery", text, err)
		}
	}

	if len(movies.texts) != 0 {
		t.Fatalf("Search() sent %v to the source, want nothing", movies.texts)
	}

	if _, err := search.Search(t.Context(), app.SearchQuery{Text: " du ", Limit: 20}); err != nil ||
		!reflect.DeepEqual(movies.texts, []string{"du"}) {
		t.Errorf("Search(\" du \") = %v with texts %v, want the text du", err, movies.texts)
	}

	got, err := app.NewListMovies(movies, beforeDocumentaries).
		List(t.Context(), app.MovieQuery{Status: domain.NowShowing, Limit: 5})
	if err != nil || len(got.Movies) != 5 || got.Total != 25 ||
		got.Movies[0].Code.String() != "movie-0" {
		t.Errorf(
			"List() = %d movies, total %d, error %v, want the first 5 of 25",
			len(got.Movies),
			got.Total,
			err,
		)
	}
}

// beforeDocumentaries is 2026-10-25 14:00 in Tbilisi, the day before the premiere of
// new-year.
func beforeDocumentaries() time.Time {
	return time.Date(2026, 10, 25, 14, 0, 0, 0, kernel.TbilisiZone())
}

// listedMovies answers fixed lists of movies.
type listedMovies struct {
	showing, soon, found []domain.Movie
}

func (l listedMovies) Movies(_ context.Context, status domain.MovieStatus) (app.MovieList, error) {
	if status == domain.ComingSoon {
		return app.MovieList{Movies: l.soon}, nil
	}

	return app.MovieList{Movies: l.showing}, nil
}

func (l listedMovies) Search(context.Context, string) (app.MovieList, error) {
	return app.MovieList{Movies: l.found}, nil
}

func premiered(t *testing.T, code, premiere string, status domain.MovieStatus) domain.Movie {
	t.Helper()

	return domain.Movie{
		ID:       code + "-id",
		Code:     mustMovieCode(t, code),
		Status:   status,
		Premiere: premiere,
	}
}

func codesOf(movies []domain.Movie) []string {
	codes := make([]string, 0, len(movies))
	for _, movie := range movies {
		codes = append(codes, movie.Code.String()+" "+string(movie.Status))
	}

	return codes
}

// premiereSource lists a movie with the premiere of today, one with the premiere of
// tomorrow, and one with a year only. The search finds the one of tomorrow.
func premiereSource(t *testing.T) listedMovies {
	t.Helper()

	return listedMovies{
		showing: []domain.Movie{
			premiered(t, "today", "2026-10-25", domain.NowShowing),
			premiered(t, "new-year", "2026-10-26", domain.NowShowing),
			premiered(t, "a-year", "2026", domain.NowShowing),
		},
		soon:  []domain.Movie{premiered(t, "dune-part-three", "2026", domain.ComingSoon)},
		found: []domain.Movie{premiered(t, "new-year", "", domain.NowShowing)},
	}
}

// CAVCINE-SC-028: A movie now showing with a premiere after today is coming soon, after
// the movies of Cavea coming soon. A premiere of today or a year keeps the status.
func TestAMovieBeforeItsPremiereIsComingSoon(t *testing.T) {
	t.Parallel()

	source := premiereSource(t)
	list := app.NewListMovies(source, beforeDocumentaries)

	tests := []struct {
		name string
		call func() (app.Movies, error)
		want []string
	}{
		{
			"now showing",
			func() (app.Movies, error) {
				return list.List(t.Context(), app.MovieQuery{Status: domain.NowShowing})
			},
			[]string{"today now_showing", "a-year now_showing"},
		},
		{
			"coming soon",
			func() (app.Movies, error) {
				return list.List(t.Context(), app.MovieQuery{Status: domain.ComingSoon})
			},
			[]string{"dune-part-three coming_soon", "new-year coming_soon"},
		},
		{
			"search",
			func() (app.Movies, error) {
				return app.NewSearchMovies(source, beforeDocumentaries).
					Search(t.Context(), app.SearchQuery{Text: "new"})
			},
			[]string{"new-year coming_soon"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.call()
			if err != nil {
				t.Fatalf("error = %v", err)
			}

			if codes := codesOf(
				got.Movies,
			); !reflect.DeepEqual(codes, tt.want) ||
				got.Total != len(tt.want) {
				t.Errorf("movies = %v of %d, want %v", codes, got.Total, tt.want)
			}
		})
	}

	if source.showing[1].Status != domain.NowShowing {
		t.Error("ListMovies changed the list of its source")
	}
}
