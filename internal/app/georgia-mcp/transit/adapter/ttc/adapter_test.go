package ttc_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/adapter/ttc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	ttcclient "github.com/abgeo/georgia-mcp/pkg/ttc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata   = "../../../../../../pkg/ttc/testdata"
	testKey    = "secret-test-key"
	route385   = "1:R29925"
	metroLine1 = "1:Metro_Metro_1"
	park       = "1:1000"
	akhmeteli  = "1:metro_1_1"
)

func readAt() time.Time {
	return time.Date(2026, 10, 19, 22, 10, 24, 0, kernel.TbilisiZone())
}

// syncBuffer is a log sink that the handler writes from many goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p) //nolint:wrapcheck // A bytes.Buffer never fails.
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// reply is a status and a body that the fake TTC sends for one path.
type reply struct {
	status int
	body   string
}

// files maps a path and a selector of the query to a file of testdata.
func files() map[string]string {
	return map[string]string{
		"/v2/stops en":                             "stops-en.json",
		"/v2/stops ka":                             "stops-ka.json",
		"/v3/routes en":                            "routes-en.json",
		"/v3/routes ka":                            "routes-ka.json",
		"/v3/routes/1:R29925 en":                   "route-385.json",
		"/v3/routes/1:R29925 ka":                   "route-385-ka.json",
		"/v3/routes/1:Metro_Metro_1 en":            "route-metro-1-en.json",
		"/v3/routes/1:Metro_Metro_1 ka":            "route-metro-1-ka.json",
		"/v3/routes/1:R29925/stops 1:01":           "pattern-stops-385.json",
		"/v3/routes/1:Metro_Metro_1/stops 1:01":    "pattern-stops-metro-1-1.json",
		"/v3/routes/1:Metro_Metro_1/stops 0:01":    "pattern-stops-metro-1-0.json",
		"/v3/routes/1:R29925/schedule 1:01":        "schedule-385.json",
		"/v3/routes/1:Metro_Metro_1/schedule 1:01": "schedule-metro-1.json",
		"/v2/stops/1:1000/arrival-times ":          "arrival-times.json",
		"/v2/stops/1:1000/arrival-times 1:R29925":  "arrival-times-385.json",
		"/v2/stops/1:metro_1_1/arrival-times ":     "arrival-times-metro.json",
		"/v2/plan ":                                "plan.json",
		"/v2/geocode ":                             "geocode.json",
	}
}

// selector picks the part of the query that chooses the file of a path.
func selector(r *http.Request) string {
	query := r.URL.Query()

	switch {
	case query.Has("patternSuffix"):
		return query.Get("patternSuffix")
	case strings.HasSuffix(r.URL.Path, "/arrival-times"):
		return query.Get("routeId")
	case r.URL.Path == "/v2/plan" || r.URL.Path == "/v2/geocode":
		return ""
	default:
		return query.Get("locale")
	}
}

// fakeTTC serves the recorded answers. An override replaces the answer of a path.
func fakeTTC(t *testing.T, overrides map[string]reply) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != testKey {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))

			return
		}

		if override, found := overrides[r.URL.Path]; found {
			w.WriteHeader(override.status)
			_, _ = w.Write([]byte(override.body))

			return
		}

		name, found := files()[r.URL.Path+" "+selector(r)]
		if !found {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write(recorded(t, name))
	}))
	t.Cleanup(server.Close)

	return server
}

func recorded(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Error(err)
	}

	return body
}

func adapterWith(
	t *testing.T,
	key string,
	overrides map[string]reply,
	sink *syncBuffer,
) *ttc.Adapter {
	t.Helper()

	server := fakeTTC(t, overrides)

	return ttc.NewAdapter(
		ttcclient.New(vendorhttp.WithBaseURL(server.URL), ttcclient.WithAPIKey(key)),
		readAt,
		slog.New(slog.NewJSONHandler(sink, nil)),
	)
}

func newAdapter(t *testing.T, overrides map[string]reply) *ttc.Adapter {
	t.Helper()

	return adapterWith(t, testKey, overrides, &syncBuffer{})
}

// TTCTRANS-SC-015: The adapter joins the stops of both languages by identifier and
// gives each stop its kind.
func TestStopsJoinsBothLanguages(t *testing.T) {
	t.Parallel()

	list, err := newAdapter(t, nil).Stops(t.Context())
	if err != nil {
		t.Fatalf("Stops() error = %v", err)
	}

	byID := make(map[domain.StopID]domain.Stop, len(list.Stops))
	for _, stop := range list.Stops {
		byID[stop.ID] = stop
	}

	want := map[domain.StopID]domain.Stop{
		park: {
			ID:   park,
			Code: "1000",
			Kind: domain.KindBus,
			Name: domain.Name{
				Georgian: "ვასო გოძიაშვილის ბაღი",
				English:  "Vaso Godziashvili Park",
			},
			Location: domain.Point{Lat: 41.725095, Lon: 44.755362},
		},
		akhmeteli: {
			ID: akhmeteli, Kind: domain.KindMetro,
			Name:     byID[akhmeteli].Name,
			Location: byID[akhmeteli].Location,
		},
		"1:gondola_1": {
			ID: "1:gondola_1", Kind: domain.KindCableCar,
			Name:     domain.Name{Georgian: "რიყე", English: "Rike"},
			Location: domain.Point{Lat: 41.692039, Lon: 44.810776},
		},
	}

	for id, stop := range want {
		if diff := cmp.Diff(stop, byID[id]); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", id, diff)
		}
	}

	if !list.ReadAt.Equal(readAt()) {
		t.Errorf("ReadAt = %v, want %v", list.ReadAt, readAt())
	}
}

// TTCTRANS-SC-015: A mode that the adapter does not know fails the answer.
func TestStopsRefusesAnUnknownMode(t *testing.T) {
	t.Parallel()

	adapter := newAdapter(t, map[string]reply{"/v2/stops": {
		status: http.StatusOK,
		body:   `[{"id":"1:1","code":"1","name":"A","lat":41.7,"lon":44.8,"vehicleMode":"TRAM"}]`,
	}})

	if _, err := adapter.Stops(t.Context()); !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Stops() error = %v, want ErrSourceUnavailable", err)
	}
}

// TTCTRANS-SC-016: A route of the mode BUS with a minibus identifier is a minibus. The
// metro and the cable car keep their kinds.
func TestRoutesGivesTheKind(t *testing.T) {
	t.Parallel()

	list, err := newAdapter(t, nil).Routes(t.Context())
	if err != nil {
		t.Fatalf("Routes() error = %v", err)
	}

	kinds := make(map[domain.RouteID]domain.Kind, len(list.Routes))
	for _, route := range list.Routes {
		kinds[route.ID] = route.Kind
	}

	want := map[domain.RouteID]domain.Kind{
		"1:minibusR25521":     domain.KindMinibus,
		route385:              domain.KindBus,
		metroLine1:            domain.KindMetro,
		"1:Gondola_Gondola_1": domain.KindCableCar,
	}
	for id, kind := range want {
		if kinds[id] != kind {
			t.Errorf("%s has the kind %q, want %q", id, kinds[id], kind)
		}
	}
}

// TTCTRANS-SC-017: A live arrival takes the minutes of tracking, an arrival of the
// timetable takes the minutes of the timetable, and a negative count leaves the
// arrival out.
func TestArrivalsChoosesTheMinutes(t *testing.T) {
	t.Parallel()

	adapter := newAdapter(t, nil)

	list, err := adapter.Arrivals(
		t.Context(),
		app.ArrivalRequest{Stop: park, Language: kernel.LanguageEnglish},
	)
	if err != nil {
		t.Fatalf("Arrivals() error = %v", err)
	}

	var minibus domain.Arrival

	for _, arrival := range list.Arrivals {
		if arrival.RouteNumber == "472" {
			minibus = arrival
		}
	}

	want := domain.Arrival{
		RouteNumber: "472", Kind: domain.KindBus, Headsign: "ლობჟანიძის ქ.",
		Minutes: 5, Live: true, Expected: readAt().Add(5 * time.Minute),
	}
	if diff := cmp.Diff(want, minibus); diff != "" {
		t.Errorf("472 mismatch (-want +got):\n%s", diff)
	}

	metro, err := adapter.Arrivals(t.Context(), app.ArrivalRequest{Stop: akhmeteli})
	if err != nil || len(metro.Arrivals) != 1 || metro.Arrivals[0].Live ||
		metro.Arrivals[0].Minutes != 3 || metro.Arrivals[0].Kind != domain.KindMetro {
		t.Errorf(
			"metro = %+v, %v, want one arrival of the timetable in 3 minutes",
			metro.Arrivals,
			err,
		)
	}
}

// TTCTRANS-SC-017: An arrival of the timetable with negative minutes is left out.
func TestArrivalsLeavesOutANegativeCount(t *testing.T) {
	t.Parallel()

	adapter := newAdapter(t, map[string]reply{"/v2/stops/1:1000/arrival-times": {
		status: http.StatusOK,
		body: `[{"shortName":"385","headsign":"A","vehicleMode":"BUS","realtime":false,` +
			`"realtimeArrivalMinutes":0,"scheduledArrivalMinutes":-5},` +
			`{"shortName":"310","headsign":"B","vehicleMode":"BUS","realtime":true,` +
			`"realtimeArrivalMinutes":4,"scheduledArrivalMinutes":-131}]`,
	}})

	list, err := adapter.Arrivals(t.Context(), app.ArrivalRequest{Stop: park})
	if err != nil {
		t.Fatalf("Arrivals() error = %v", err)
	}

	if len(list.Arrivals) != 1 || list.Arrivals[0].RouteNumber != "310" ||
		list.Arrivals[0].Minutes != 4 {
		t.Errorf("arrivals = %+v, want only the 310 in 4 minutes", list.Arrivals)
	}
}

// TTCTRANS-SC-018: A route of two patterns gives two directions, with the headsign in
// both languages and the stops in order.
func TestDirectionsOfTheMetro(t *testing.T) {
	t.Parallel()

	metro, err := newAdapter(t, nil).Directions(t.Context(), metroLine1)
	if err != nil {
		t.Fatalf("Directions(metro) error = %v", err)
	}

	type digest struct {
		ID       int
		English  string
		Georgian bool
		Stops    int
	}

	got := make([]digest, 0, len(metro.Directions))
	for _, direction := range metro.Directions {
		got = append(got, digest{
			ID:       direction.ID,
			English:  direction.Headsign.English,
			Georgian: direction.Headsign.Georgian != "",
			Stops:    len(direction.Stops),
		})
	}

	want := []digest{
		{ID: 1, English: "Varketili", Georgian: true, Stops: 16},
		{ID: 0, English: "Akhmeteli Theatre", Georgian: true, Stops: 16},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Directions(metro) mismatch (-want +got):\n%s", diff)
	}
}

// TTCTRANS-SC-018: A loop gives one direction whose first stop is also its last.
func TestDirectionsOfALoop(t *testing.T) {
	t.Parallel()

	loop, err := newAdapter(t, nil).Directions(t.Context(), route385)
	if err != nil {
		t.Fatalf("Directions(385) error = %v", err)
	}

	if len(loop.Directions) != 1 {
		t.Fatalf("Directions(385) = %+v, want one direction", loop.Directions)
	}

	stops := loop.Directions[0].Stops
	if len(stops) != 22 || stops[0] != "1:1001" || stops[21] != "1:1001" {
		t.Errorf("stops = %v, want 22 from and to 1:1001", stops)
	}
}

// TTCTRANS-SC-019: A timetable gives one day for each date of the week. A direction
// that the route lacks is refused. A time after midnight keeps its hour.
func TestTimetable(t *testing.T) {
	t.Parallel()

	adapter := newAdapter(t, nil)

	read, err := adapter.Timetable(t.Context(), route385, 1)
	if err != nil {
		t.Fatalf("Timetable(385, 1) error = %v", err)
	}

	first, last := read.Timetable.Span()
	if len(read.Timetable.Days) != 7 || first.String() != "2026-10-19" ||
		last.String() != "2026-10-25" {
		t.Errorf(
			"Timetable(385, 1) has %d days from %s to %s, want 7 from 2026-10-19 to 2026-10-25",
			len(read.Timetable.Days),
			first,
			last,
		)
	}

	if _, err := adapter.Timetable(
		t.Context(),
		route385,
		0,
	); !errors.Is(
		err,
		app.ErrUnknownDirection,
	) {
		t.Errorf("Timetable(385, 0) error = %v, want ErrUnknownDirection", err)
	}

	metro, err := adapter.Timetable(t.Context(), metroLine1, 1)
	if err != nil {
		t.Fatalf("Timetable(metro, 1) error = %v", err)
	}

	date, _ := kernel.ParseDate("2026-10-19")

	times, found := metro.Timetable.Times(date, "1:metro_1_10")
	if !found || times[len(times)-1].String() != "24:10" {
		t.Errorf("the last time of 1:metro_1_10 = %v, want 24:10", times)
	}
}

// TTCTRANS-SC-020: The instants of a plan are in Tbilisi. A 400 is no trip. A 500 is
// an outage.
func TestPlan(t *testing.T) {
	t.Parallel()

	request := app.TripRequest{
		From:       domain.Point{Lat: 41.7251, Lon: 44.7554},
		To:         domain.Point{Lat: 41.7151, Lon: 44.8271},
		Timing:     domain.TimingLeaveNow,
		Kinds:      domain.RouteKinds(),
		Preference: domain.PreferenceFastest,
	}

	plan, err := newAdapter(t, nil).Plan(t.Context(), request)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	ride := plan.Itineraries[0].Legs[1]
	if got := ride.Start.Format(time.RFC3339); got != "2026-10-19T21:42:51+04:00" {
		t.Errorf("the first ride starts at %s, want 2026-10-19T21:42:51+04:00", got)
	}

	if ride.Kind != domain.KindBus || ride.RouteNumber != "310" || ride.Stops != 12 ||
		ride.From.Name != "Vaso Godziashvili Park" {
		t.Errorf("the first ride = %+v, want the 310 of 12 stops from Vaso Godziashvili Park", ride)
	}

	for status, want := range map[int]error{
		http.StatusBadRequest:          app.ErrTripNotFound,
		http.StatusInternalServerError: kernel.ErrSourceUnavailable,
	} {
		adapter := newAdapter(t, map[string]reply{"/v2/plan": {status: status}})
		if _, err := adapter.Plan(t.Context(), request); !errors.Is(err, want) {
			t.Errorf("Plan() with %d error = %v, want %v", status, err, want)
		}
	}
}

// TTCTRANS-SC-021: A refused key logs one error that names the variable and holds no
// key, and the error is an outage.
func TestARefusedKeyIsLogged(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterWith(t, "wrong-key", nil, sink)

	_, err := adapter.Stops(t.Context())
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Stops() error = %v, want ErrSourceUnavailable", err)
	}

	log := sink.String()
	if strings.Count(log, `"msg":"the TTC refused the key"`) != 1 ||
		!strings.Contains(log, `"variable":"GEOMCP_TRANSIT_TTC_API_KEY"`) ||
		!strings.Contains(log, `"level":"ERROR"`) {
		t.Errorf("log = %s, want one error record that names the variable", log)
	}

	if strings.Contains(log, "wrong-key") {
		t.Errorf("log = %s, want no key", log)
	}
}

// TTCTRANS-SC-022: A place outside Tbilisi is left out. A place with no name takes the
// street and the house number.
func TestPlaces(t *testing.T) {
	t.Parallel()

	adapter := newAdapter(t, map[string]reply{"/v2/geocode": {
		status: http.StatusOK,
		body: `{"features":[` +
			`{"geometry":{"coordinates":[40,40]},"properties":{"name":"Far"}},` +
			`{"geometry":{"coordinates":[44.7997,41.6966]},"properties":{"street":"Shota Rustaveli Avenue",` +
			`"housenumber":"17","locality":"Mtatsminda"}},` +
			`{"geometry":{"coordinates":[44.79]},"properties":{"name":"Broken"}}]}`,
	}})

	list, err := adapter.Places(t.Context(), "rustaveli", kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Places() error = %v", err)
	}

	want := []domain.Place{{
		Name: "Shota Rustaveli Avenue 17", Street: "Shota Rustaveli Avenue", HouseNumber: "17",
		District: "Mtatsminda", Location: domain.Point{Lat: 41.6966, Lon: 44.7997},
	}}
	if diff := cmp.Diff(want, list.Places); diff != "" {
		t.Errorf("Places() mismatch (-want +got):\n%s", diff)
	}

	recordedList, err := newAdapter(t, nil).Places(t.Context(), "rustaveli", kernel.LanguageEnglish)
	if err != nil || len(recordedList.Places) == 0 || recordedList.Places[0].Name != "Rustaveli" {
		t.Errorf("Places() = %+v, %v, want Rustaveli first", recordedList.Places, err)
	}
}
