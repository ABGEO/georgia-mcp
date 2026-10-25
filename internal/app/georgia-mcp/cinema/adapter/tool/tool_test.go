package tool_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	listCinemas  = "cinema_list_cinemas"
	listMovies   = "cinema_list_movies"
	searchMovies = "cinema_search_movies"
	listSessions = "cinema_list_sessions"
	getSeatMap   = "cinema_get_seat_map"

	keyLanguage = "language"
	keyMovie    = "movie"
	keyDate     = "date"
	keySession  = "session"

	keyName     = "name"
	keyQuery    = "query"
	galleriaKA  = "კავეა გალერია"
	galleriaEN  = "Cavea Galleria"
	fighterKA   = "ქუჩის მებრძოლი"
	fighterEN   = "Street Fighter"
	sessionCode = "cag_5/27-10-2026/22:15"
	vendorBody  = "an unexpected error has occurred"
)

func now() time.Time {
	return time.Date(2026, 10, 25, 14, 0, 0, 0, kernel.TbilisiZone())
}

func mustParse[T any](t *testing.T, parse func(string) (T, error), text string) T {
	t.Helper()

	value, err := parse(text)
	if err != nil {
		t.Fatal(err)
	}

	return value
}

func mustPrice(t *testing.T, text string) domain.Price {
	t.Helper()

	return mustParse(t, domain.NewPrice, text)
}

// errVendor is a failure of no known kind, with the text of a vendor body.
var errVendor = errors.New(vendorBody)

// fakeCavea answers one cinema, one movie, and one session at 22:15 on 2026-10-27, or
// fails each call with err.
type fakeCavea struct {
	t   *testing.T
	err error
}

func (f fakeCavea) Cinemas(context.Context) (app.CinemaList, error) {
	return app.CinemaList{Cinemas: []domain.Cinema{{
		ID:      "galleria-id",
		Code:    mustParse(f.t, domain.ParseCinemaCode, "galleria"),
		Name:    domain.Text{Georgian: galleriaKA, English: galleriaEN},
		City:    "Tbilisi",
		Address: "2/4 Rustaveli Ave",
	}}, ReadAt: now()}, f.err
}

func (f fakeCavea) Movies(_ context.Context, status domain.MovieStatus) (app.MovieList, error) {
	if status == domain.ComingSoon {
		return app.MovieList{Movies: []domain.Movie{}, ReadAt: now()}, f.err
	}

	return app.MovieList{Movies: []domain.Movie{f.movie()}, ReadAt: now()}, f.err
}

func (f fakeCavea) Search(context.Context, string) (app.MovieList, error) {
	movie := f.movie()

	return app.MovieList{Movies: []domain.Movie{{
		ID: movie.ID, Code: movie.Code, Status: movie.Status, Name: movie.Name,
		Duration: movie.Duration, AgeRating: movie.AgeRating,
	}}, ReadAt: now()}, f.err
}

func (f fakeCavea) Sessions(context.Context) (app.SessionList, error) {
	return app.SessionList{
		Sessions: []domain.Session{f.session(map[domain.SeatClass]domain.Price{
			domain.Standard: mustPrice(f.t, "20"),
		})},
		ReadAt: now(),
	}, f.err
}

func (f fakeCavea) MovieSessions(context.Context, string) (app.SessionList, error) {
	return app.SessionList{
		Sessions: []domain.Session{f.session(map[domain.SeatClass]domain.Price{
			domain.Economy:  mustPrice(f.t, "20"),
			domain.Standard: mustPrice(f.t, "20"),
			domain.VIP:      mustPrice(f.t, "29"),
			domain.Status:   mustPrice(f.t, "22"),
		})},
		ReadAt: now(),
	}, f.err
}

func (f fakeCavea) Occupancy(
	_ context.Context,
	codes []domain.SessionCode,
) (map[domain.SessionCode]domain.OccupancyLevel, time.Time, error) {
	levels := map[domain.SessionCode]domain.OccupancyLevel{}
	for _, code := range codes {
		levels[code] = domain.ManySeats
	}

	return levels, now(), f.err
}

func (f fakeCavea) SeatMap(
	context.Context,
	domain.SessionCode,
) ([]domain.Seat, time.Time, error) {
	return []domain.Seat{
		{Row: "1", Number: 2, Class: domain.VIP, SpecialNeeds: true},
		{Row: "1", Number: 1, Class: domain.Standard, Taken: true},
		{Row: "2", Number: 1, Class: domain.Standard},
	}, now(), f.err
}

func (f fakeCavea) movie() domain.Movie {
	return domain.Movie{
		ID:     "street-fighter-id",
		Code:   mustParse(f.t, domain.ParseMovieCode, "street-fighter"),
		Status: domain.NowShowing,
		Name:   domain.Text{Georgian: fighterKA, English: fighterEN},
		Genres: domain.TextList{
			Georgian: []string{"მძაფრსიუჟეტიანი"},
			English:  []string{"Action"},
		},
		Duration:    127 * time.Minute,
		AgeRating:   "PG-13",
		Premiere:    "2026-10-15",
		Directors:   domain.TextList{English: []string{"Kitao Sakurai"}},
		Description: domain.Text{English: "A fight."},
	}
}

func (f fakeCavea) session(prices map[domain.SeatClass]domain.Price) domain.Session {
	subtitles := domain.AudioGeorgian

	return domain.Session{
		Code:      mustParse(f.t, domain.ParseSessionCode, sessionCode),
		MovieID:   "street-fighter-id",
		CinemaID:  "galleria-id",
		Hall:      "5",
		Audio:     domain.AudioEnglish,
		Subtitles: &subtitles,
		Prices:    prices,
	}
}

func connect(t *testing.T, source fakeCavea) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		ListCinemas:  app.NewListCinemas(source),
		ListMovies:   app.NewListMovies(source, now),
		SearchMovies: app.NewSearchMovies(source, now),
		ListSessions: app.NewListSessions(
			source,
			source,
			source,
			source,
			now,
			slog.New(slog.DiscardHandler),
		),
		GetSeatMap: app.NewGetSeatMap(source, source, source, source, now),
	}); err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = registry.RunStdio(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).
		Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

func callTool(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}

	return result
}

// output calls the tool and returns its output as JSON values.
func output(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) map[string]any {
	t.Helper()

	result := callTool(t, session, name, args)
	if result.IsError {
		t.Fatalf("CallTool(%s) = %+v, want a result", name, result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

func errorText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()

	result := callTool(t, session, name, args)
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool(%s) = %+v, want a tool error", name, result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool(%s) content = %T, want text", name, result.Content[0])
	}

	return text.Text
}

func decode(t *testing.T, text string) map[string]any {
	t.Helper()

	var value map[string]any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatal(err)
	}

	return value
}

// at follows a path of keys and indexes through JSON values.
func at(value any, path ...any) any {
	for _, step := range path {
		switch key := step.(type) {
		case string:
			object, _ := value.(map[string]any)
			value = object[key]
		case int:
			list, _ := value.([]any)
			if key >= len(list) {
				return nil
			}

			value = list[key]
		}
	}

	return value
}

// nameCase is a call of one tool and the path to a name in its output.
type nameCase struct {
	tool        string
	args        map[string]any
	path        []any
	ka, english string
}

func nameCases() []nameCase {
	return []nameCase{
		{listCinemas, map[string]any{}, []any{"cinemas", 0, keyName}, galleriaKA, galleriaEN},
		{listMovies, map[string]any{}, []any{"movies", 0, keyName}, fighterKA, fighterEN},
		{
			searchMovies,
			map[string]any{keyQuery: "street"},
			[]any{"movies", 0, keyName},
			fighterKA, fighterEN,
		},
		{
			listSessions,
			map[string]any{keyMovie: "street-fighter", keyDate: "2026-10-27"},
			[]any{"sessions", 0, "cinema_name"},
			galleriaKA, galleriaEN,
		},
		{
			getSeatMap,
			map[string]any{keySession: sessionCode},
			[]any{"session", "movie_name"},
			fighterKA, fighterEN,
		},
	}
}

// CAVCINE-SC-019: Each tool with ka gives the Georgian names, and with no language the
// English names. ru gives the message of the language.
func TestEachToolGivesTheNamesInTheLanguage(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeCavea{t: t})

	for _, tt := range nameCases() {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()

			english := output(t, session, tt.tool, tt.args)

			georgian := maps.Clone(tt.args)
			georgian[keyLanguage] = "ka"

			if got := at(output(t, session, tt.tool, georgian), tt.path...); got != tt.ka {
				t.Errorf("%s with ka = %v, want %s", tt.tool, got, tt.ka)
			}

			if got := at(english, tt.path...); got != tt.english {
				t.Errorf("%s with no language = %v, want %s", tt.tool, got, tt.english)
			}

			georgian[keyLanguage] = "ru"

			want := `language: "ru" is not a language of this tool. Use ka or en.`
			if got := errorText(t, session, tt.tool, georgian); got != want {
				t.Errorf("%s with ru = %q, want %q", tt.tool, got, want)
			}
		})
	}
}

// CAVCINE-SC-019: The sessions and the seat map in English hold each field of the tools.
// An empty list of sessions holds no note of the occupancy.
func TestSessionsAndSeatMapInEnglish(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeCavea{t: t})

	scheduled := `{"code":"cag_5/27-10-2026/22:15","movie_code":"street-fighter",
		"movie_name":"` + fighterEN + `","cinema_code":"galleria","cinema_name":"` + galleriaEN + `",
		"hall":"5","start":"2026-10-27T22:15:00+04:00","audio_language":"en",
		"subtitle_language":"ka","formats":[],"prices":[{"class":"economy","price":"20"},
		{"class":"standard","price":"20"},{"class":"vip","price":"29"},
		{"class":"status","price":"22"}]`

	wantSessions := decode(t, `{"source":"Cavea","as_of":"2026-10-25T14:00:00+04:00",
		"stale":false,"occupancy_as_of":"2026-10-25T14:00:00+04:00","total":1,
		"sessions":[`+scheduled+`,"occupancy":"many_seats"}]}`)

	got := output(
		t,
		session,
		listSessions,
		map[string]any{keyMovie: "street-fighter", keyDate: "2026-10-27"},
	)
	if diff := cmp.Diff(wantSessions, got); diff != "" {
		t.Errorf("cinema_list_sessions mismatch (-want +got):\n%s", diff)
	}

	wantMap := decode(t, `{"source":"Cavea","as_of":"2026-10-25T14:00:00+04:00",
		"session":`+scheduled+`},
		"classes":[{"class":"standard","price":"20","seats":2,"free":1},
		{"class":"vip","price":"29","seats":1,"free":1}],
		"rows":[{"row":"1","seats":[{"number":1,"class":"standard","taken":true},
		{"number":2,"class":"vip","taken":false,"special_needs":true}]},
		{"row":"2","seats":[{"number":1,"class":"standard","taken":false}]}]}`)

	got = output(t, session, getSeatMap, map[string]any{keySession: sessionCode})
	if diff := cmp.Diff(wantMap, got); diff != "" {
		t.Errorf("cinema_get_seat_map mismatch (-want +got):\n%s", diff)
	}

	empty := output(t, session, listSessions, map[string]any{keyDate: "2026-10-26"})

	wantEmpty := decode(t, `{"source":"Cavea","as_of":"2026-10-25T14:00:00+04:00",
		"stale":false,"total":0,"sessions":[]}`)
	if diff := cmp.Diff(wantEmpty, empty); diff != "" {
		t.Errorf("cinema_list_sessions of a date with no session mismatch (-want +got):\n%s", diff)
	}

	movie := output(t, session, listMovies, map[string]any{})
	if url := at(movie, "movies", 0, "url"); url != "https://cavea.ge/movie/street-fighter" {
		t.Errorf("url = %v, want https://cavea.ge/movie/street-fighter", url)
	}
}

// messageCase is a call of one tool and the message of its error.
type messageCase struct {
	tool string
	args map[string]any
	want string
}

func messageCases() []messageCase {
	return append([]messageCase{
		{
			listMovies,
			map[string]any{"status": "archived"},
			`status: "archived" is not a status. Use now_showing or coming_soon.`,
		},
		{
			searchMovies,
			map[string]any{keyQuery: "a"},
			"query: give 2 to 100 characters, in English.",
		},
		{listMovies, map[string]any{"limit": 500}, "limit: 500 is out of range. Use 1 to 100."},
		{
			getSeatMap,
			map[string]any{keySession: "xyz"},
			`session: "xyz" is not a session code. Use a code from cinema_list_sessions, ` +
				`for example cag_5/27-10-2026/22:15.`,
		},
		{
			getSeatMap,
			map[string]any{keySession: "cag_5/29-10-2026/22:15"},
			`session: "cag_5/29-10-2026/22:15" is not a session of Cavea. Use cinema_list_sessions.`,
		},
		{
			getSeatMap,
			map[string]any{keySession: "cag_5/24-10-2026/22:15"},
			`session: "cag_5/24-10-2026/22:15" started. Use cinema_list_sessions for a later session.`,
		},
	}, sessionMessageCases()...)
}

func sessionMessageCases() []messageCase {
	return []messageCase{
		{listSessions, map[string]any{"limit": -1}, "limit: -1 is out of range. Use 1 to 100."},
		{
			listSessions,
			map[string]any{keyDate: "27.10"},
			`date: "27.10" is not a date. Use YYYY-MM-DD.`,
		},
		{
			listSessions,
			map[string]any{keyDate: "2026-10-24"},
			"date: 2026-10-24 is before today. Use today or a later date.",
		},
		{
			listSessions,
			map[string]any{"audio_language": "ru"},
			`audio_language: "ru" is not a language of Cavea. Use ka or en.`,
		},
		{
			listSessions,
			map[string]any{"format": "4dx"},
			`format: "4dx" is not a format. Use 3d, imax, dolby_atmos, screenx, or infinity_vision.`,
		},
		{
			listSessions,
			map[string]any{keyMovie: "x"},
			`movie: "x" is not a movie of Cavea. Use cinema_list_movies or cinema_search_movies.`,
		},
		{
			listSessions,
			map[string]any{keyMovie: "Not A Code"},
			`movie: "Not A Code" is not a movie of Cavea. Use cinema_list_movies or cinema_search_movies.`,
		},
		{
			listSessions,
			map[string]any{"cinema": "x"},
			`cinema: "x" is not a cinema of Cavea. Use cinema_list_cinemas.`,
		},
	}
}

// CAVCINE-SC-027: Each condition of the table of errors gives its message.
func TestEachConditionGivesItsMessage(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeCavea{t: t})

	for _, tt := range messageCases() {
		t.Run(fmt.Sprintf("%s %v", tt.tool, tt.args), func(t *testing.T) {
			t.Parallel()

			if got := errorText(t, session, tt.tool, tt.args); got != tt.want {
				t.Errorf("%s %v = %q, want %q", tt.tool, tt.args, got, tt.want)
			}
		})
	}
}

// CAVCINE-SC-027: A failure of Cavea gives the message of an outage, and any other
// failure the message of the server. Neither holds the text of Cavea.
func TestFailuresHideTheTextOfCavea(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{
			fmt.Errorf("calling Cavea: %w: %w", kernel.ErrSourceUnavailable, errVendor),
			"Cavea did not answer. Try again in a few minutes.",
		},
		{errVendor, "The server failed to read the data of Cavea."},
	}

	calls := []struct {
		tool string
		args map[string]any
	}{
		{listCinemas, map[string]any{}},
		{listMovies, map[string]any{}},
		{searchMovies, map[string]any{keyQuery: "street"}},
		{listSessions, map[string]any{}},
		{getSeatMap, map[string]any{keySession: sessionCode}},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			session := connect(t, fakeCavea{t: t, err: tt.err})

			for _, call := range calls {
				t.Run(call.tool, func(t *testing.T) {
					t.Parallel()

					got := errorText(t, session, call.tool, call.args)
					if got != tt.want || strings.Contains(got, vendorBody) {
						t.Errorf("%s with %v = %q, want %q", call.tool, tt.err, got, tt.want)
					}
				})
			}
		})
	}
}
