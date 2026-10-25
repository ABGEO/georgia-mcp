package cavea_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/cavea"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	sessionCode    = "cag_5/27-10-2026/22:15"
	escapedSession = "cag_5%2F27-10-2026%2F22:15"
	streetFighter  = "6ab27d3fde5280679e341878"
)

// recorder serves the recorded answers of Cavea by path, and keeps each request.
type recorder struct {
	mu       sync.Mutex
	requests []*http.Request
	answers  map[string]string
}

func (r *recorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	r.requests = append(r.requests, req.Clone(req.Context()))
	r.mu.Unlock()

	key := req.Method + " " + req.URL.EscapedPath()
	if req.URL.Path == "/movies" && req.URL.Query().Get("filter") == "coming-soon" {
		key += "?coming-soon"
	}

	file, ok := r.answers[key]
	if !ok {
		http.NotFound(w, req)

		return
	}

	if strings.HasPrefix(key, "POST /api/booking/seat-ticket/") {
		w.Header().Add("Set-Cookie", "cavea_vid=visitor; Max-Age=31536000; Path=/; Secure")
		w.Header().Add("Set-Cookie", "__Host-cavea_vproof=proof; Max-Age=3600; Path=/; HttpOnly")
	}

	path := "testdata/" + file

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(body)
}

func (r *recorder) received() []*http.Request {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.requests
}

func recorded() map[string]string {
	return map[string]string{
		"GET /api/theaters":                               "theaters.json",
		"GET /api/theaters/hall-map":                      "hall-map.json",
		"GET /movies":                                     "movies.rsc",
		"GET /movies?coming-soon":                         "movies-coming-soon.rsc",
		"GET /api/movies/search":                          "search-street.json",
		"GET /sessions":                                   "sessions.rsc",
		"GET /api/booking/sessions/all/" + streetFighter:  "sessions-all-street-fighter.json",
		"GET /api/sessions/availability":                  "availability.json",
		"POST /api/booking/seat-ticket/" + escapedSession: "seat-ticket.json",
		"GET /api/booking/seats/" + escapedSession:        "seats-cag5.json",
	}
}

func newClient(t *testing.T, answers map[string]string) (*cavea.Client, *recorder) {
	t.Helper()

	rec := &recorder{answers: answers}
	server := httptest.NewServer(rec)
	t.Cleanup(server.Close)

	return cavea.New(vendorhttp.WithBaseURL(server.URL)), rec
}

// serveBody answers every request with one body and one status.
func serveBody(t *testing.T, body string, status int) *cavea.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return cavea.New(vendorhttp.WithBaseURL(server.URL))
}

// calls returns one call of each method of the client.
func calls() map[string]func(ctx context.Context, client *cavea.Client) error {
	return map[string]func(ctx context.Context, client *cavea.Client) error{
		"Theaters": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.Theaters(ctx)

			return err
		},
		"HallMap": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.HallMap(ctx)

			return err
		},
		"Movies": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.Movies(ctx, false)

			return err
		},
		"SearchMovies": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.SearchMovies(ctx, "street")

			return err
		},
		"Sessions": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.Sessions(ctx)

			return err
		},
		"MovieSessions": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.MovieSessions(ctx, streetFighter)

			return err
		},
		"Availability": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.Availability(ctx, []string{sessionCode})

			return err
		},
		"SeatMap": func(ctx context.Context, client *cavea.Client) error {
			_, err := client.SeatMap(ctx, sessionCode)

			return err
		},
	}
}

// CAVCINE-SC-001: Each method sends the User-Agent of the server, the Origin of the site
// of Cavea, and the header X-Requested-With of its pages.
func TestEveryMethodSendsTheHeaders(t *testing.T) {
	t.Parallel()

	for name, call := range calls() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, rec := newClient(t, recorded())
			if err := call(t.Context(), client); err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}

			for _, req := range rec.received() {
				agent := req.Header.Get("User-Agent")
				if !strings.HasPrefix(agent, "georgia-mcp/") ||
					!strings.Contains(agent, "https://github.com/abgeo/georgia-mcp") {
					t.Errorf("%s %s: User-Agent = %q", req.Method, req.URL.Path, agent)
				}

				if origin := req.Header.Get("Origin"); origin != "https://cavea.ge" {
					t.Errorf(
						"%s %s: Origin = %q, want https://cavea.ge",
						req.Method,
						req.URL.Path,
						origin,
					)
				}

				if with := req.Header.Get("X-Requested-With"); with != "cavea-web" {
					t.Errorf(
						"%s %s: X-Requested-With = %q, want cavea-web",
						req.Method,
						req.URL.Path,
						with,
					)
				}
			}
		})
	}
}

// CAVCINE-SC-002: The client reads the 8 theaters of the recorded answer. Galleria has no
// field is_active.
func TestTheatersReadsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, recorded())

	theaters, err := client.Theaters(t.Context())
	if err != nil {
		t.Fatalf("Theaters() error = %v", err)
	}

	if len(theaters) != 8 {
		t.Fatalf("Theaters() holds %d theaters, want 8", len(theaters))
	}

	galleria := theaters[4]

	want := cavea.Theater{
		ID:       "5a1583c9956167428b722f81",
		Name:     "კავეა გალერია",
		NameEng:  "Cavea Galleria",
		City:     "Tbilisi",
		Address:  "2/4 Rustaveli Avenue",
		IsCinema: true,
	}
	if !reflect.DeepEqual(galleria, want) {
		t.Errorf("Theaters()[4] = %+v, want %+v", galleria, want)
	}
}

// CAVCINE-SC-003: Movies reads the data stream of the page of movies, now showing and
// coming soon.
func TestMoviesReadsTheStream(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, recorded())

	showing, err := client.Movies(t.Context(), false)
	if err != nil {
		t.Fatalf("Movies(false) error = %v", err)
	}

	soon, err := client.Movies(t.Context(), true)
	if err != nil {
		t.Fatalf("Movies(true) error = %v", err)
	}

	if len(showing) != 25 || len(soon) != 10 {
		t.Fatalf("Movies() hold %d and %d movies, want 25 and 10", len(showing), len(soon))
	}

	clayface := showing[0]

	want := []string{
		"clayface-01",
		"Clayface",
		"108",
		"R",
		"DC, Horror, Sci-Fi, Thriller",
		"22.10.2026",
	}

	got := []string{
		clayface.Slug, clayface.NameEnglish, strconv.Itoa(clayface.Duration),
		clayface.MPAARating, clayface.GenreEng, clayface.Premiere,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Movies(false)[0] = %+v, want %+v", got, want)
	}
}

// CAVCINE-SC-003: Movies asks for the data stream, and for the movies coming soon with the
// filter of the page.
func TestMoviesSendsTheHeadersOfTheStream(t *testing.T) {
	t.Parallel()

	client, rec := newClient(t, recorded())

	for _, comingSoon := range []bool{false, true} {
		if _, err := client.Movies(t.Context(), comingSoon); err != nil {
			t.Fatalf("Movies(%t) error = %v", comingSoon, err)
		}
	}

	requests := rec.received()
	for _, req := range requests {
		if req.Header.Get("Rsc") != "1" || req.Header.Get("Accept") != "text/x-component" {
			t.Errorf("%s: RSC = %q, Accept = %q, want 1 and text/x-component",
				req.URL, req.Header.Get("Rsc"), req.Header.Get("Accept"))
		}
	}

	if requests[0].URL.RawQuery != "" || requests[1].URL.RawQuery != "filter=coming-soon" {
		t.Errorf("queries = %q and %q, want none and filter=coming-soon",
			requests[0].URL.RawQuery, requests[1].URL.RawQuery)
	}
}

// CAVCINE-SC-003: A stream with no list of movies, and a list that does not decode, are
// errors.
func TestMoviesMirrorsTheStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want error
	}{
		{name: "no list", body: `0:{"children":"x"}`, want: cavea.ErrNoData},
		{name: "malformed", body: `6:{"movies":[{"id":`, want: cavea.ErrMalformedData},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			movies, err := serveBody(t, tt.body, http.StatusOK).Movies(t.Context(), false)
			if !errors.Is(err, tt.want) || movies != nil {
				t.Errorf("Movies() = %v, %v, want %v", movies, err, tt.want)
			}
		})
	}
}

// CAVCINE-SC-004: SearchMovies sends the text and reads the movies that Cavea matches.
func TestSearchMoviesSendsTheText(t *testing.T) {
	t.Parallel()

	client, rec := newClient(t, recorded())

	movies, err := client.SearchMovies(t.Context(), "street")
	if err != nil {
		t.Fatalf("SearchMovies() error = %v", err)
	}

	if query := rec.received()[0].URL.Query().Get("q"); query != "street" {
		t.Errorf("q = %q, want street", query)
	}

	slugs := make([]string, 0, len(movies))
	for _, movie := range movies {
		slugs = append(slugs, movie.Slug)
	}

	want := []string{"the-end-of-oak-street", "street-fighter"}
	if !reflect.DeepEqual(slugs, want) {
		t.Errorf("SearchMovies() slugs = %+v, want %+v", slugs, want)
	}
}

// CAVCINE-SC-005: Sessions reads the 3 date groups of the data stream of the page of
// sessions.
func TestSessionsReadsTheStream(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, recorded())

	groups, err := client.Sessions(t.Context())
	if err != nil {
		t.Fatalf("Sessions() error = %v", err)
	}

	counts := map[string]int{}

	for _, group := range groups {
		for _, movie := range group.Movies {
			counts[group.Date] += len(movie.Sessions)
		}
	}

	want := map[string]int{"25-10-2026": 23, "26-10-2026": 152, "27-10-2026": 152}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("Sessions() counts = %+v, want %+v", counts, want)
	}

	first := groups[0].Movies[0]

	wantFirst := cavea.PageSession{
		ID:              "cav_5/25-10-2026/22:00",
		Cinema:          "54ec23a69561676764d1a778",
		SessionLanguage: "geo",
		Price:           "18",
	}

	if first.Movie.Slug != "verity-01" {
		t.Errorf("the first movie = %+v, want verity-01", first.Movie)
	}

	if !reflect.DeepEqual(first.Sessions[0], wantFirst) {
		t.Errorf("the first session = %+v, want %+v", first.Sessions[0], wantFirst)
	}
}

// CAVCINE-SC-006: MovieSessions decodes the text of the data, and the text of the prices
// of each session.
func TestMovieSessionsDecodesTheTexts(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, recorded())

	sessions, err := client.MovieSessions(t.Context(), streetFighter)
	if err != nil {
		t.Fatalf("MovieSessions() error = %v", err)
	}

	if len(sessions) != 31 {
		t.Fatalf("MovieSessions() holds %d sessions, want 31", len(sessions))
	}

	want := cavea.MovieSession{
		ID:              sessionCode,
		Cinema:          "5a1583c9956167428b722f81",
		SessionLanguage: "geo",
		Prices:          map[string]string{"eco": "20", "std": "20", "vip": "29", "sta": "22"},
	}
	if !reflect.DeepEqual(sessions[0], want) {
		t.Errorf("MovieSessions()[0] = %+v, want %+v", sessions[0], want)
	}
}

// CAVCINE-SC-006: An answer with the status error is an error, not an empty list.
func TestAnAnswerWithTheStatusErrorIsAnError(t *testing.T) {
	t.Parallel()

	client := serveBody(t, `{"status":"error","message":"Movie not found"}`, http.StatusOK)

	sessions, err := client.MovieSessions(t.Context(), streetFighter)
	if !errors.Is(err, cavea.ErrNotSuccess) || sessions != nil {
		t.Errorf("MovieSessions() = %v, %v, want ErrNotSuccess", sessions, err)
	}
}

// CAVCINE-SC-007: Availability refuses more than 30 codes with no call, and reads the
// level of each code.
func TestAvailabilityReadsTheLevels(t *testing.T) {
	t.Parallel()

	client, rec := newClient(t, recorded())

	many := make([]string, cavea.MaxAvailabilityIDs+1)
	for i := range many {
		many[i] = "cag_5/27-10-2026/" + strconv.Itoa(10+i%10) + ":00"
	}

	if _, err := client.Availability(t.Context(), many); !errors.Is(err, cavea.ErrTooManySessions) {
		t.Errorf("Availability(31 codes) error = %v, want ErrTooManySessions", err)
	}

	if len(rec.received()) != 0 {
		t.Fatalf("Availability(31 codes) sent %d calls, want none", len(rec.received()))
	}

	levels, err := client.Availability(t.Context(), []string{sessionCode, "cep_1/27-10-2026/13:30"})
	if err != nil {
		t.Fatalf("Availability() error = %v", err)
	}

	if ids := rec.received()[0].URL.Query().
		Get("ids"); ids != sessionCode+",cep_1/27-10-2026/13:30" {
		t.Errorf("ids = %q, want the two codes with a comma", ids)
	}

	if !reflect.DeepEqual(levels, map[string]string{sessionCode: "green"}) {
		t.Errorf("Availability() = %+v, want %+v", levels, map[string]string{sessionCode: "green"})
	}
}

// CAVCINE-SC-008: SeatMap asks for a ticket, then sends the ticket and the cookies of the
// ticket answer with the request of the seats.
func TestSeatMapSendsTheTicketAndTheCookies(t *testing.T) {
	t.Parallel()

	client, rec := newClient(t, recorded())

	seats, err := client.SeatMap(t.Context(), sessionCode)
	if err != nil {
		t.Fatalf("SeatMap() error = %v", err)
	}

	requests := rec.received()
	if len(requests) != 2 || requests[0].Method != http.MethodPost ||
		requests[1].Method != http.MethodGet {
		t.Fatalf("SeatMap() sent %d calls, want a POST then a GET", len(requests))
	}

	seat := requests[1]
	if ticket := seat.Header.Get("X-Seat-Ticket"); ticket != "test-ticket" {
		t.Errorf("X-Seat-Ticket = %q, want test-ticket", ticket)
	}

	if cookie := seat.Header.Get(
		"Cookie",
	); cookie != "cavea_vid=visitor; __Host-cavea_vproof=proof" {
		t.Errorf("Cookie = %q, want the two cookies of the ticket answer", cookie)
	}

	if len(seats) != 91 {
		t.Fatalf("SeatMap() holds %d seats, want 91", len(seats))
	}

	want := cavea.Seat{ID: "cag5_r6_s1", Row: "6", Seat: 1, Class: "vip", IsTaken: true}
	if !reflect.DeepEqual(seats[0], want) {
		t.Errorf("SeatMap()[0] = %+v, want %+v", seats[0], want)
	}
}

// CAVCINE-SC-008: A ticket answer with no ticket is an error, and the client sends no
// request of the seats.
func TestSeatMapWithNoTicketSendsNoRequestOfTheSeats(t *testing.T) {
	t.Parallel()

	answers := recorded()
	answers["POST /api/booking/seat-ticket/"+escapedSession] = "availability.json"

	client, rec := newClient(t, answers)

	seats, err := client.SeatMap(t.Context(), sessionCode)
	if !errors.Is(err, cavea.ErrNoTicket) || seats != nil {
		t.Errorf("SeatMap() = %v, %v, want ErrNoTicket", seats, err)
	}

	if len(rec.received()) != 1 {
		t.Errorf("SeatMap() sent %d calls, want the POST only", len(rec.received()))
	}
}

// CAVCINE-SC-009: A status that is not 200 is an APIError that wraps ErrUnexpectedStatus,
// on each method.
func TestEveryMethodReturnsAnAPIErrorOnAForbiddenStatus(t *testing.T) {
	t.Parallel()

	for name, call := range calls() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := serveBody(t, `{"status":"error","message":"Forbidden"}`, http.StatusForbidden)

			err := call(t.Context(), client)

			apiErr, ok := errors.AsType[*cavea.APIError](err)
			if !ok || apiErr.StatusCode != http.StatusForbidden {
				t.Fatalf("%s() error = %v, want an *APIError with 403", name, err)
			}

			if !errors.Is(err, cavea.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
			}
		})
	}
}
