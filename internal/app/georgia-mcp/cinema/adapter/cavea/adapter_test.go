package cavea_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/adapter/cavea"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	caveaclient "github.com/abgeo/georgia-mcp/pkg/cavea"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata      = "../../../../../../pkg/cavea/testdata"
	streetFighter = "6ab27d3fde5280679e341878"
	galleria2215  = "cag_5/27-10-2026/22:15"
)

func readAt() time.Time {
	return time.Date(2026, 10, 25, 14, 0, 0, 0, time.UTC)
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// vendor is a fake Cavea that serves a body for each path and counts the calls.
type vendor struct {
	bodies map[string][]byte

	mu    sync.Mutex
	calls map[string]int
}

func (v *vendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.URL.Query().Get("filter") == "coming-soon" {
		path += "?coming-soon"
	}

	v.mu.Lock()
	v.calls[path]++
	v.mu.Unlock()

	body, ok := v.bodies[path]
	if !ok {
		http.NotFound(w, r)

		return
	}

	if strings.HasPrefix(path, "/api/booking/seat-ticket/") {
		w.Header().Add("Set-Cookie", "cavea_vid=visitor; Path=/")
	}

	_, _ = w.Write(body)
}

func (v *vendor) count(path string) int {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.calls[path]
}

func newVendor(t *testing.T) *vendor {
	t.Helper()

	return &vendor{
		calls: map[string]int{},
		bodies: map[string][]byte{
			"/api/theaters":          readFile(t, "theaters.json"),
			"/api/theaters/hall-map": readFile(t, "hall-map.json"),
			"/movies":                readFile(t, "movies.rsc"),
			"/movies?coming-soon":    readFile(t, "movies-coming-soon.rsc"),
			"/api/movies/search":     readFile(t, "search-street.json"),
			"/sessions":              readFile(t, "sessions.rsc"),
			"/api/booking/sessions/all/" + streetFighter: readFile(
				t,
				"sessions-all-street-fighter.json",
			),
			"/api/sessions/availability":               readFile(t, "availability.json"),
			"/api/booking/seat-ticket/" + galleria2215: readFile(t, "seat-ticket.json"),
			"/api/booking/seats/" + galleria2215:       readFile(t, "seats-cag5.json"),
		},
	}
}

func adapterOf(t *testing.T, fake *vendor) *cavea.Adapter {
	t.Helper()

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	return cavea.NewAdapter(caveaclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

func mustSessionCode(t *testing.T, text string) domain.SessionCode {
	t.Helper()

	code, err := domain.ParseSessionCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return code
}

// CAVCINE-SC-011: The open cinemas get a code from the English name, with the word Cavea
// removed.
func TestCinemasGivesTheOpenCinemas(t *testing.T) {
	t.Parallel()

	list, err := adapterOf(t, newVendor(t)).Cinemas(t.Context())
	if err != nil {
		t.Fatalf("Cinemas() error = %v", err)
	}

	got := make([]string, 0, len(list.Cinemas))
	for _, cinema := range list.Cinemas {
		got = append(got, cinema.Code.String())
	}

	want := []string{"tbilisi_mall", "east_point", "galleria", "city_mall_saburtalo", "grand_mall"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Cinemas() codes = %v, want %v", got, want)
	}

	eastPoint := list.Cinemas[1]

	wantEast := domain.Cinema{
		ID:      "55e0d0da956167a5a79cc43c",
		Code:    eastPoint.Code,
		Name:    domain.Text{Georgian: "კავეა ისთ ფოინთი", English: "Cavea East Point"},
		City:    "Tbilisi",
		Address: "2 Aleksandre Tvalchrelidze St",
	}
	if !reflect.DeepEqual(eastPoint, wantEast) || !list.ReadAt.Equal(readAt()) {
		t.Errorf("Cinemas()[1] = %+v, want %+v", eastPoint, wantEast)
	}

	if _, offset := list.ReadAt.Zone(); offset != 4*60*60 {
		t.Errorf("ReadAt = %v, want the zone of Tbilisi", list.ReadAt)
	}
}

// CAVCINE-SC-012: The premiere is a date or a year, and the genres of clayface-01 keep
// their order in each language.
func TestMoviesGivesThePremiere(t *testing.T) {
	t.Parallel()

	adapter := adapterOf(t, newVendor(t))

	showing, err := adapter.Movies(t.Context(), domain.NowShowing)
	if err != nil {
		t.Fatalf("Movies(now_showing) error = %v", err)
	}

	clayface := showing.Movies[0]

	want := []any{
		"clayface-01", "Clayface", domain.NowShowing, 108 * time.Minute, "2026-10-22",
		4, "საშინელებათა", 4, "Horror",
	}

	got := []any{
		clayface.Code.String(), clayface.Name.English, clayface.Status, clayface.Duration,
		clayface.Premiere, len(clayface.Genres.Georgian), clayface.Genres.Georgian[1],
		len(clayface.Genres.English), clayface.Genres.English[1],
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Movies(now_showing)[0] = %v, want %v", got, want)
	}

	soon, err := adapter.Movies(t.Context(), domain.ComingSoon)
	if err != nil {
		t.Fatalf("Movies(coming_soon) error = %v", err)
	}

	guria := soon.Movies[0]

	wantSoon := []any{"guria-01", "2026", domain.ComingSoon}
	if gotSoon := []any{
		guria.Code.String(),
		guria.Premiere,
		guria.Status,
	}; !reflect.DeepEqual(
		gotSoon,
		wantSoon,
	) {
		t.Errorf("Movies(coming_soon)[0] = %v, want %v", gotSoon, wantSoon)
	}
}

func moviesBySlug(t *testing.T) map[string]domain.Movie {
	t.Helper()

	showing, err := adapterOf(t, newVendor(t)).Movies(t.Context(), domain.NowShowing)
	if err != nil {
		t.Fatalf("Movies(now_showing) error = %v", err)
	}

	bySlug := map[string]domain.Movie{}
	for _, movie := range showing.Movies {
		bySlug[movie.Code.String()] = movie
	}

	return bySlug
}

// CAVCINE-SC-012: Each language splits its list on its own, at each comma, with no space
// at the ends of an item. An English field in Georgian letters is empty.
func TestMoviesSplitsEachLanguage(t *testing.T) {
	t.Parallel()

	bySlug := moviesBySlug(t)

	tests := []struct {
		name string
		got  any
		want any
	}{
		{
			"street-fighter actors",
			bySlug["street-fighter"].Actors,
			domain.TextList{
				Georgian: []string{"ჯეისონ მომოა", "დევიდ დასტმალჩიანი", "ნოა სენტინეო"},
				English: []string{
					"Jason Momoa",
					"David Dastmalchian",
					"Noah Centineo",
					"Olivier Richters",
				},
			},
		},
		{
			"digger-01 English actors", bySlug["digger-01"].Actors.English,
			[]string{
				"Tom Cruise", "Riz Ahmed", "John Goodman", "Sandra Hüller",
				"Michael Stuhlbarg", "Jesse Plemons",
			},
		},
		{
			"minions-monsters English genres", bySlug["minions-monsters"].Genres.English,
			[]string{"Adventure", "Animation", "Comedy", "Crime Family", "Sci-Fi"},
		},
		{
			"beri-gabrieli genres", bySlug["beri-gabrieli"].Genres,
			domain.TextList{Georgian: []string{"დრამა"}},
		},
		{
			"beri-gabrieli name", bySlug["beri-gabrieli"].Name,
			domain.Text{Georgian: "ბრძოლა - ბერი გაბრიელი"},
		},
		{"moana-01 English description", bySlug["moana-01"].Description.English, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tt.want, tt.got); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tt.name, diff)
			}
		})
	}
}

// CAVCINE-SC-013: The hall comes from the hall map, and a page session has the price of
// the class Standard.
func TestSessionsGivesTheHallAndThePrice(t *testing.T) {
	t.Parallel()

	fake := newVendor(t)

	list, err := adapterOf(t, fake).Sessions(t.Context())
	if err != nil {
		t.Fatalf("Sessions() error = %v", err)
	}

	halls := hallsOf(list.Sessions)
	first := list.Sessions[0]

	want := []any{327, "IMAX", "10", 1, "18", "6a958854de52800b99417502", domain.AudioGeorgian}

	got := []any{
		len(list.Sessions), halls["cep_1"], halls["ccm_x"], len(first.Prices),
		first.Prices[domain.Standard].String(), first.MovieID, first.Audio,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sessions() = %v, want %v", got, want)
	}

	fake.bodies["/api/theaters/hall-map"] = []byte(strings.Replace(
		string(fake.bodies["/api/theaters/hall-map"]), `"ccm_x":"10",`, "", 1))

	again, err := adapterOf(t, fake).Sessions(t.Context())
	if err != nil {
		t.Fatalf("Sessions() error = %v", err)
	}

	if hall := hallsOf(again.Sessions)["ccm_x"]; hall != "X" {
		t.Errorf("hall of ccm_x with no entry in the hall map = %q, want X", hall)
	}
}

func hallsOf(sessions []domain.Session) map[string]string {
	halls := map[string]string{}
	for _, session := range sessions {
		halls[session.Code.HallKey()] = session.Hall
	}

	return halls
}

// CAVCINE-SC-013: A session of a movie has the price of each class, and its formats.
func TestMovieSessionsGivesEveryPrice(t *testing.T) {
	t.Parallel()

	list, err := adapterOf(t, newVendor(t)).MovieSessions(t.Context(), streetFighter)
	if err != nil {
		t.Fatalf("MovieSessions() error = %v", err)
	}

	first := list.Sessions[0]

	prices := map[domain.SeatClass]string{}
	for class, price := range first.Prices {
		prices[class] = price.String()
	}

	want := []any{31, galleria2215, "5", streetFighter, map[domain.SeatClass]string{
		domain.Economy: "20", domain.Standard: "20", domain.VIP: "29", domain.Status: "22",
	}}
	if got := []any{
		len(list.Sessions),
		first.Code.String(),
		first.Hall,
		first.MovieID,
		prices,
	}; !reflect.DeepEqual(
		got,
		want,
	) {
		t.Errorf("MovieSessions() = %v, want %v", got, want)
	}

	imax := map[string][]domain.Format{}
	for _, session := range list.Sessions {
		imax[session.Code.HallKey()] = session.Formats
	}

	if !reflect.DeepEqual(imax["cep_1"], []domain.Format{domain.FormatIMAX}) {
		t.Errorf("formats of cep_1 = %v, want imax", imax["cep_1"])
	}
}

// CAVCINE-SC-014: A session in another language stays out. A seat of an unknown class
// and a price that is not above zero fail the answer.
func TestValuesOfCavea(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		replace [2]string
		want    int
		wantErr error
	}{
		{
			name:    "audio language",
			replace: [2]string{`\"session_language\": \"geo\"`, `\"session_language\": \"rus\"`},
			want:    30,
		},
		{
			name:    "subtitle language",
			replace: [2]string{`\"subtitle_language\": null`, `\"subtitle_language\": \"rus\"`},
			want:    30,
		},
		{
			name:    "price",
			replace: [2]string{`\\\"vip\\\": \\\"29\\\"`, `\\\"vip\\\": \\\"0\\\"`},
			wantErr: kernel.ErrSourceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := newVendor(t)
			replaceBody(t, fake, "/api/booking/sessions/all/"+streetFighter, tt.replace)

			list, err := adapterOf(t, fake).MovieSessions(t.Context(), streetFighter)
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && len(list.Sessions) != tt.want) {
				t.Errorf("MovieSessions() = %d sessions, %v, want %d and %v",
					len(list.Sessions), err, tt.want, tt.wantErr)
			}
		})
	}
}

// CAVCINE-SC-014: A seat of a class that the context does not know fails the seat map.
func TestSeatMapRefusesAnUnknownClass(t *testing.T) {
	t.Parallel()

	fake := newVendor(t)
	replaceBody(
		t,
		fake,
		"/api/booking/seats/"+galleria2215,
		[2]string{`"ttype":"vip"`, `"ttype":"gold"`},
	)

	if _, _, err := adapterOf(
		t,
		fake,
	).SeatMap(t.Context(), mustSessionCode(t, galleria2215)); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("SeatMap() error = %v, want kernel.ErrSourceUnavailable", err)
	}
}

func replaceBody(t *testing.T, fake *vendor, path string, replace [2]string) {
	t.Helper()

	changed := strings.Replace(string(fake.bodies[path]), replace[0], replace[1], 1)
	if changed == string(fake.bodies[path]) {
		t.Fatalf("the answer of %s holds no %s", path, replace[0])
	}

	fake.bodies[path] = []byte(changed)
}

// CAVCINE-SC-017: The levels of 70 sessions take 3 calls of 30 codes at most, and map the
// colors of Cavea to the three levels.
func TestOccupancySendsOneCallForEachThirtyCodes(t *testing.T) {
	t.Parallel()

	fake := newVendor(t)
	fake.bodies["/api/sessions/availability"] = []byte(
		`{"status":"success","data":{"cag_5/27-10-2026/10:00":"green",` +
			`"cag_5/27-10-2026/10:01":"yellow","cag_5/27-10-2026/10:02":"red"}}`)

	codes := make([]domain.SessionCode, 70)
	for index := range codes {
		codes[index] = mustSessionCode(
			t,
			fmt.Sprintf("cag_5/27-10-2026/%02d:%02d", 10+index/60, index%60),
		)
	}

	levels, _, err := adapterOf(t, fake).Occupancy(t.Context(), codes)
	if err != nil {
		t.Fatalf("Occupancy() error = %v", err)
	}

	if calls := fake.count("/api/sessions/availability"); calls != 3 {
		t.Errorf("Occupancy() sent %d calls, want 3", calls)
	}

	want := map[domain.SessionCode]domain.OccupancyLevel{
		codes[0]: domain.ManySeats, codes[1]: domain.FillingUp, codes[2]: domain.AlmostSoldOut,
	}
	if !reflect.DeepEqual(levels, want) {
		t.Errorf("Occupancy() = %v, want %v", levels, want)
	}
}

// CAVCINE-SC-018: The seat map gives each seat of the hall with its class and its state.
func TestSeatMapGivesTheSeats(t *testing.T) {
	t.Parallel()

	seats, _, err := adapterOf(
		t,
		newVendor(t),
	).SeatMap(t.Context(), mustSessionCode(t, galleria2215))
	if err != nil {
		t.Fatalf("SeatMap() error = %v", err)
	}

	taken := 0

	for _, seat := range seats {
		if seat.Taken {
			taken++
		}
	}

	want := domain.Seat{Row: "6", Number: 1, Class: domain.VIP, Taken: true}
	if len(seats) != 91 || taken != 28 || seats[0] != want {
		t.Errorf(
			"SeatMap() = %d seats, %d taken, first %+v, want 91, 28, %+v",
			len(seats),
			taken,
			seats[0],
			want,
		)
	}
}
