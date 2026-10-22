package ttc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/ttc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testKey    = "test-key"
	english    = "en"
	localeKey  = "locale"
	modeBus    = "BUS"
	route385   = "1:R29925"
	pattern385 = "1:01"
	tbilisiBox = "44.66221063,41.84319686,45.050854455,41.59149226"
	quick      = "quick"
	rustaveli  = "Rustaveli"
)

// recorder serves one file of testdata with 200 and keeps the last request.
type recorder struct {
	server  *httptest.Server
	request *http.Request
}

func serveFile(t *testing.T, path string) *recorder {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	rec := &recorder{}
	rec.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.request = r.Clone(r.Context())

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(rec.server.Close)

	return rec
}

func (r *recorder) client() *ttc.Client {
	return ttc.New(vendorhttp.WithBaseURL(r.server.URL), ttc.WithAPIKey(testKey))
}

func checkPath(t *testing.T, request *http.Request, want string) {
	t.Helper()

	if got := request.URL.EscapedPath(); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func checkQuery(t *testing.T, request *http.Request, want map[string]string) {
	t.Helper()

	query := request.URL.Query()
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("query %s = %q, want %q", key, got, value)
		}
	}
}

// TTCTRANS-SC-001: The client sends the path, the locale, the key, the Accept header,
// and the User-Agent.
func TestStopsSendsTheRequest(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/stops-en.json")

	if _, err := rec.client().Stops(t.Context(), english); err != nil {
		t.Fatalf("Stops() error = %v", err)
	}

	checkPath(t, rec.request, "/v2/stops")
	checkQuery(t, rec.request, map[string]string{localeKey: english})

	headers := map[string]string{"X-Api-Key": testKey, "Accept": "application/json"}
	for name, want := range headers {
		if got := rec.request.Header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	agent := rec.request.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-ttc/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-ttc/<version> (+<repository>)", agent)
	}
}

// TTCTRANS-SC-001: The client decodes a stop with a code and a station with none.
func TestStopsDecodesTheStops(t *testing.T) {
	t.Parallel()

	stops, err := serveFile(t, "testdata/stops-en.json").client().Stops(t.Context(), english)
	if err != nil {
		t.Fatalf("Stops() error = %v", err)
	}

	byID := make(map[string]ttc.Stop, len(stops))
	for _, stop := range stops {
		byID[stop.ID] = stop
	}

	park := byID["1:1000"]
	if park.Code == nil || *park.Code != "1000" || park.Name != "Vaso Godziashvili Park" ||
		park.VehicleMode != modeBus {
		t.Errorf("1:1000 = %+v, want the code 1000, Vaso Godziashvili Park, BUS", park)
	}

	if gondola, ok := byID["1:gondola_1"]; !ok || gondola.Code != nil ||
		gondola.VehicleMode != "GONDOLA" {
		t.Errorf("1:gondola_1 = %+v, want no code and GONDOLA", gondola)
	}
}

// TTCTRANS-SC-002: The client asks every mode, and decodes a minibus route as a bus.
func TestRoutesAsksEveryMode(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/routes-en.json")

	routes, err := rec.client().Routes(t.Context(), english)
	if err != nil {
		t.Fatalf("Routes() error = %v", err)
	}

	checkPath(t, rec.request, "/v3/routes")
	checkQuery(t, rec.request, map[string]string{"modes": "BUS,SUBWAY,GONDOLA", localeKey: english})

	index := 0
	for index < len(routes) && routes[index].ID != "1:minibusR25521" {
		index++
	}

	if index == len(routes) || routes[index].ShortName != "472" || routes[index].Mode != modeBus {
		t.Errorf("routes = %+v, want 1:minibusR25521 with the number 472 and BUS", routes)
	}
}

// TTCTRANS-SC-003: The client sends the path of a route.
func TestRouteSendsThePath(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/route-385.json")

	route, err := rec.client().Route(t.Context(), route385, english)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	checkPath(t, rec.request, "/v3/routes/1:R29925")

	if route.ShortName != "385" || len(route.Patterns) != 1 || route.Patterns[0].DirectionID != 1 {
		t.Errorf("Route() = %+v, want 385 with one pattern of direction 1", route)
	}
}

// TTCTRANS-SC-003: The client sends the path and the pattern of the stops of a route.
func TestPatternStopsSendsThePattern(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/pattern-stops-385.json")

	stops, err := rec.client().PatternStops(t.Context(), route385, pattern385, english)
	if err != nil {
		t.Fatalf("PatternStops() error = %v", err)
	}

	checkPath(t, rec.request, "/v3/routes/1:R29925/stops")
	checkQuery(t, rec.request, map[string]string{"patternSuffix": pattern385, localeKey: english})

	if len(stops) != 22 {
		t.Errorf("PatternStops() gives %d stops, want 22", len(stops))
	}
}

// TTCTRANS-SC-003: The client sends the path and the pattern of a timetable.
func TestScheduleSendsThePattern(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/schedule-385.json")

	groups, err := rec.client().Schedule(t.Context(), route385, pattern385, english)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	checkPath(t, rec.request, "/v3/routes/1:R29925/schedule")
	checkQuery(t, rec.request, map[string]string{"patternSuffix": pattern385, localeKey: english})

	if len(groups) != 3 || groups[0].FromDay != "MONDAY" || len(groups[0].ServiceDates) != 5 ||
		!strings.HasPrefix(groups[0].Stops[0].ArrivalTimes, "6:45,6:55") {
		t.Errorf("Schedule() = %+v, want 3 groups from MONDAY at 6:45", groups)
	}
}

// TTCTRANS-SC-004: With no route, the client sends no route and no maximum, and asks
// the arrivals of the timetable.
func TestArrivalTimesWithNoRoute(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/arrival-times.json")

	arrivals, err := rec.client().ArrivalTimes(
		t.Context(), "1:1000", ttc.ArrivalTimesQuery{Locale: english},
	)
	if err != nil {
		t.Fatalf("ArrivalTimes() error = %v", err)
	}

	checkPath(t, rec.request, "/v2/stops/1:1000/arrival-times")
	checkQuery(t, rec.request, map[string]string{
		localeKey: english, "ignoreScheduledArrivalTimes": "false",
	})

	query := rec.request.URL.Query()
	if query.Has("routeId") || query.Has("maxNumberOfArrivalTimes") {
		t.Errorf(
			"query = %q, want no routeId and no maxNumberOfArrivalTimes",
			rec.request.URL.RawQuery,
		)
	}

	want := ttc.ArrivalTime{
		ShortName:               "472",
		Color:                   "0033B4",
		Headsign:                "ლობჟანიძის ქ.",
		PatternSuffix:           "1:03",
		VehicleMode:             modeBus,
		Realtime:                true,
		RealtimeArrivalMinutes:  5,
		ScheduledArrivalMinutes: -119,
	}
	if len(arrivals) != 12 || arrivals[2] != want {
		t.Errorf("arrivals = %+v, want 12 with %+v third", arrivals, want)
	}
}

// TTCTRANS-SC-004: With a route, the client sends the route and the maximum.
func TestArrivalTimesWithARoute(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/arrival-times-385.json")

	if _, err := rec.client().ArrivalTimes(
		t.Context(), "1:1000", ttc.ArrivalTimesQuery{Locale: english, RouteID: route385, Max: 5},
	); err != nil {
		t.Fatalf("ArrivalTimes() error = %v", err)
	}

	checkQuery(t, rec.request, map[string]string{
		"routeId": route385, "maxNumberOfArrivalTimes": "5",
	})
}

// TTCTRANS-SC-005: The client sends the points, the timing, the modes, and the
// preference, and decodes the itineraries.
func TestPlanSendsTheQuery(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/plan.json")

	plan, err := rec.client().Plan(t.Context(), ttc.PlanQuery{
		FromLat: 41.7251, FromLon: 44.7554, ToLat: 41.7151, ToLon: 44.8271,
		DepartMode: "departAt", Date: "2026-10-20", Time: "09:00",
		Modes: []string{"WALK", modeBus}, Optimize: quick, Locale: english,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	checkPath(t, rec.request, "/v2/plan")
	checkQuery(t, rec.request, map[string]string{
		"fromPlace": "41.7251,44.7554", "toPlace": "41.7151,44.8271",
		"departMode": "departAt", "date": "2026-10-20", "time": "09:00",
		"modes": "WALK,BUS", "optimize": quick, localeKey: english,
	})

	if len(plan.Itineraries) != 3 {
		t.Fatalf("Plan() gives %d itineraries, want 3", len(plan.Itineraries))
	}

	ride := plan.Itineraries[0].Legs[1]
	if ride.Mode != modeBus || ride.Route == nil || ride.Route.ShortName != "310" ||
		ride.StartTime != "2026-10-19T17:42:51.000+00:00" || len(ride.IntermediateStops) != 11 {
		t.Errorf("first ride = %+v, want the 310 at 17:42:51 UTC with 11 intermediate stops", ride)
	}
}

// TTCTRANS-SC-005: A plan to leave now sends no date and no time.
func TestPlanToLeaveNowSendsNoDate(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/plan.json")

	if _, err := rec.client().Plan(t.Context(), ttc.PlanQuery{
		FromLat: 41.7251, FromLon: 44.7554, ToLat: 41.7151, ToLon: 44.8271,
		DepartMode: "leaveNow", Modes: []string{"WALK"}, Optimize: quick, Locale: english,
	}); err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if query := rec.request.URL.Query(); query.Has("date") || query.Has("time") {
		t.Errorf("query = %q, want no date and no time", rec.request.URL.RawQuery)
	}
}

// TTCTRANS-SC-006: The client sends the text, the locale, and the box, and decodes the
// coordinates of a feature.
func TestGeocodeSendsTheQuery(t *testing.T) {
	t.Parallel()

	rec := serveFile(t, "testdata/geocode.json")

	places, err := rec.client().Geocode(t.Context(), ttc.GeocodeQuery{
		Query: rustaveli, Locale: english, BBox: tbilisiBox,
	})
	if err != nil {
		t.Fatalf("Geocode() error = %v", err)
	}

	checkPath(t, rec.request, "/v2/geocode")
	checkQuery(t, rec.request, map[string]string{
		"query": rustaveli, localeKey: english, "bbox": tbilisiBox,
	})

	first := places.Features[0]
	if len(first.Geometry.Coordinates) != 2 || first.Geometry.Coordinates[0] != 44.7896467 ||
		first.Geometry.Coordinates[1] != 41.703497 || first.Properties.Name != rustaveli {
		t.Errorf("first feature = %+v, want Rustaveli at [44.7896467, 41.703497]", first)
	}
}

// TTCTRANS-SC-007: A status other than 200 is an APIError with the status and the body.
func TestAStatusIsAnAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
	}))
	t.Cleanup(server.Close)

	_, err := ttc.New(vendorhttp.WithBaseURL(server.URL)).Stops(t.Context(), english)

	apiErr, ok := errors.AsType[*ttc.APIError](err)
	if !ok {
		t.Fatalf("Stops() error = %v, want *APIError", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Body != "Unauthorized" {
		t.Errorf("APIError = %+v, want 401 with Unauthorized", apiErr)
	}

	if !errors.Is(err, ttc.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
