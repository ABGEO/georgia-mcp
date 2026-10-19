package tool_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	searchStops  = "transit_search_stops"
	nearbyStops  = "transit_list_nearby_stops"
	listArrivals = "transit_list_arrivals"
	searchRoutes = "transit_search_routes"
	getRoute     = "transit_get_route"
	getTimetable = "transit_get_timetable"
	planTrip     = "transit_plan_trip"
	searchPlaces = "transit_search_places"
	vendorBody   = "an unexpected error has occurred"
	keyLat       = "lat"
	keyLon       = "lon"
	keyQuery     = "query"
	keyFrom      = "from"
	keyTo        = "to"
	keyRoute     = "route"
	keyStop      = "stop"
	keyTiming    = "timing"
	keyKinds     = "kinds"
)

// errVendor is a failure of no known kind, with the text of a vendor body.
var errVendor = errors.New(vendorBody)

// failing answers every use case with one error.
type failing struct {
	err error
}

func (f failing) Search(context.Context, app.StopSearch) (app.StopMatches, error) {
	return app.StopMatches{}, f.err
}

func (f failing) List(context.Context, app.NearbyQuery) (app.NearbyStops, error) {
	return app.NearbyStops{}, f.err
}

func (f failing) Get(context.Context, app.RouteQuery) (app.RouteView, error) {
	return app.RouteView{}, f.err
}

func (f failing) Plan(context.Context, app.TripQuery) (app.Trip, error) {
	return app.Trip{}, f.err
}

type failingArrivals struct{ err error }

func (f failingArrivals) List(context.Context, app.ArrivalQuery) (app.Arrivals, error) {
	return app.Arrivals{}, f.err
}

type failingRoutes struct{ err error }

func (f failingRoutes) Search(context.Context, app.RouteSearch) (app.RouteMatches, error) {
	return app.RouteMatches{}, f.err
}

type failingTimetable struct{ err error }

func (f failingTimetable) Get(context.Context, app.TimetableQuery) (app.TimetableView, error) {
	return app.TimetableView{}, f.err
}

type failingPlaces struct{ err error }

func (f failingPlaces) Search(context.Context, app.PlaceQuery) (app.Places, error) {
	return app.Places{}, f.err
}

func connect(t *testing.T, err error) *mcp.ClientSession {
	t.Helper()

	uses := tool.UseCases{
		SearchStops:  failing{err: err},
		NearbyStops:  failing{err: err},
		Arrivals:     failingArrivals{err: err},
		SearchRoutes: failingRoutes{err: err},
		Route:        failing{err: err},
		Timetable:    failingTimetable{err: err},
		Trip:         failing{err: err},
		Places:       failingPlaces{err: err},
	}

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, uses); err != nil {
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

func errorText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}

	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool(%s) = %+v, want a tool error", name, result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T, want text", result.Content[0])
	}

	return text.Text
}

type errorCase struct {
	name string
	tool string
	args map[string]any
	err  error
	want string
}

func date(text string) kernel.Date {
	parsed, err := kernel.ParseDate(text)
	if err != nil {
		panic(err)
	}

	return parsed
}

func inputCases() []errorCase {
	return []errorCase{
		{
			name: "language",
			tool: searchStops,
			args: map[string]any{keyQuery: "a", "language": "ru"},
			want: `language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{
			name: "empty query", tool: searchStops, args: map[string]any{keyQuery: "  "},
			want: "query: give 1 to 100 characters.",
		},
		{
			name: "long query",
			tool: searchPlaces,
			args: map[string]any{keyQuery: strings.Repeat("a", 101)},
			want: "query: give 1 to 100 characters.",
		},
		{
			name: "limit", tool: searchStops, args: map[string]any{keyQuery: "a", "limit": 500},
			want: "limit: 500 is out of range. Use 1 to 50.",
		},
		{
			name: "radius",
			tool: nearbyStops,
			args: map[string]any{keyLat: 41.7, keyLon: 44.8, "radius_m": 5000},
			want: "radius_m: 5000 is out of range. Use 1 to 2000.",
		},
		{
			name: "kind of stop",
			tool: searchStops,
			args: map[string]any{keyQuery: "a", "kind": "minibus"},
			want: `kind: "minibus" is not a kind of stop. Use one of: bus, metro, cable_car.`,
		},
		{
			name: "kind of route",
			tool: searchRoutes,
			args: map[string]any{"kind": "tram"},
			want: `kind: "tram" is not a kind of route. Use one of: bus, minibus, metro, cable_car.`,
		},
	}
}

func tripInputCases() []errorCase {
	from := map[string]any{keyLat: 41.7251, keyLon: 44.7554}
	to := map[string]any{keyLat: 41.7151, keyLon: 44.8271}

	return []errorCase{
		{
			name: "point",
			tool: planTrip,
			args: map[string]any{keyFrom: map[string]any{keyLat: 95, keyLon: 44.8}, keyTo: to},
			want: "from: 95, 44.8 is not a point. Use a latitude of -90 to 90 and a longitude of -180 to 180.",
		},
		{
			name: "date",
			tool: getTimetable,
			args: map[string]any{keyRoute: "385", keyStop: "1001", "date": "20/10"},
			want: `date: "20/10" is not a date. Use YYYY-MM-DD.`,
		},
		{
			name: "timing",
			tool: planTrip,
			args: map[string]any{keyFrom: from, keyTo: to, keyTiming: "now"},
			want: `timing: "now" is not valid. Use leave_now, depart_at, or arrive_by.`,
		},
		{
			name: "no time",
			tool: planTrip,
			args: map[string]any{keyFrom: from, keyTo: to, keyTiming: "depart_at"},
			want: "time: depart_at needs a time. Use HH:MM.",
		},
		{
			name: "time",
			tool: planTrip,
			args: map[string]any{keyFrom: from, keyTo: to, keyTiming: "arrive_by", "time": "9"},
			want: `time: "9" is not a time. Use HH:MM.`,
		},
		{
			name: "preference",
			tool: planTrip,
			args: map[string]any{keyFrom: from, keyTo: to, "preference": "cheap"},
			want: `preference: "cheap" is not valid. Use fastest, less_walking, or fewer_transfers.`,
		},
		{
			name: "kind of trip",
			tool: planTrip,
			args: map[string]any{keyFrom: from, keyTo: to, keyKinds: []string{"walk"}},
			want: `kinds: "walk" is not a kind of route. Use one of: bus, minibus, metro, cable_car.`,
		},
		{
			name: "minibus alone",
			tool: planTrip,
			args: map[string]any{keyFrom: from, keyTo: to, keyKinds: []string{"minibus"}},
			want: "kinds: the TTC plans bus and minibus together. Give both, or neither.",
		},
		{
			name: "bus alone",
			tool: planTrip,
			args: map[string]any{keyFrom: from, keyTo: to, keyKinds: []string{"bus", "metro"}},
			want: "kinds: the TTC plans bus and minibus together. Give both, or neither.",
		},
	}
}

func useCaseCases() []errorCase {
	metro := domain.Route{
		ID: "1:Metro_Metro_1", Number: "1", Kind: domain.KindMetro,
		Name: domain.Name{English: "M/S Akhmeteli - M/S Varketili"},
	}
	cable := domain.Route{
		ID: "1:Gondola_Gondola_1", Number: "1", Kind: domain.KindCableCar,
		Name: domain.Name{English: "Rike - Narikala"},
	}
	route385 := domain.Route{ID: "1:R29925", Number: "385", Kind: domain.KindBus}

	return []errorCase{
		{
			name: "unknown stop", tool: listArrivals, args: map[string]any{keyStop: "999999"},
			err:  &app.UnknownStopError{Text: "999999"},
			want: `stop: "999999" is not a stop of the TTC. Use transit_search_stops to find it.`,
		},
		{
			name: "unknown route", tool: getRoute, args: map[string]any{keyRoute: "999"},
			err:  &app.UnknownRouteError{Text: "999"},
			want: `route: "999" is not a route of the TTC. Use transit_search_routes to find it.`,
		},
		{
			name: "two routes", tool: getRoute, args: map[string]any{keyRoute: "1"},
			err: &app.AmbiguousRouteError{Text: "1", Routes: []domain.Route{metro, cable}},
			want: `route: "1" names 2 routes: 1 (metro, M/S Akhmeteli - M/S Varketili), ` +
				`1 (cable_car, Rike - Narikala). Give the kind.`,
		},
		{
			name: "no direction",
			tool: getTimetable,
			args: map[string]any{keyRoute: "1", keyStop: "1:metro_1_1"},
			err: &app.DirectionError{Route: metro, Directions: []domain.Direction{
				{ID: 1, Headsign: domain.Name{English: "Varketili"}},
				{ID: 0, Headsign: domain.Name{English: "Akhmeteli Theatre"}},
			}},
			want: "direction: route 1 has 2 directions: 0 (Akhmeteli Theatre), 1 (Varketili). Give one.",
		},
		{
			name: "not on the direction",
			tool: getTimetable,
			args: map[string]any{keyRoute: "385", keyStop: "1000"},
			err: &app.StopNotOnDirectionError{
				Stop:      domain.Stop{ID: "1:1000"},
				Route:     route385,
				Direction: 1,
			},
			want: `stop: "1:1000" is not on direction 1 of route 385. ` +
				"Use transit_get_route to see its stops.",
		},
	}
}

func outcomeCases() []errorCase {
	return []errorCase{
		{
			name: "outside the week",
			tool: getTimetable,
			args: map[string]any{keyRoute: "385", keyStop: "1001"},
			err: &app.DateOutsideTimetableError{
				Date: date("2026-10-26"), First: date("2026-10-19"), Last: date("2026-10-25"),
			},
			want: "date: 2026-10-26 is outside the timetable. Use a date from 2026-10-19 to 2026-10-25.",
		},
		{
			name: "outside Tbilisi",
			tool: planTrip,
			args: map[string]any{
				keyFrom: map[string]any{keyLat: 41.7, keyLon: 44.8},
				keyTo:   map[string]any{keyLat: 40, keyLon: 40},
			},
			err:  &app.OutsideTbilisiError{Field: "to", Point: domain.Point{Lat: 40, Lon: 40}},
			want: "to: 40, 40 is outside Tbilisi. The TTC serves Tbilisi only.",
		},
		{
			name: "no trip",
			tool: planTrip,
			args: map[string]any{
				keyFrom: map[string]any{keyLat: 41.7, keyLon: 44.8},
				keyTo:   map[string]any{keyLat: 41.71, keyLon: 44.81},
			},
			err:  fmt.Errorf("planning the trip: %w: %s", app.ErrTripNotFound, vendorBody),
			want: "The TTC found no trip between the two points. Try other points or another time.",
		},
		{
			name: "outage", tool: searchPlaces, args: map[string]any{keyQuery: "Rustaveli"},
			err:  fmt.Errorf("reading: %w: 500 %s", kernel.ErrSourceUnavailable, vendorBody),
			want: "The TTC did not answer. Try again in a few minutes.",
		},
		{
			name: "any other error",
			tool: searchRoutes,
			args: map[string]any{},
			err:  fmt.Errorf("decoding: %w", errVendor),
			want: "The server failed to read the data of the TTC.",
		},
	}
}

// TTCTRANS-SC-038: Each row of the errors table gives a tool error with its exact
// message, and no message holds a vendor body.
func TestErrorsGiveTheMessages(t *testing.T) {
	t.Parallel()

	for _, tt := range slices.Concat(inputCases(), tripInputCases(), useCaseCases(), outcomeCases()) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := errorText(t, connect(t, tt.err), tt.tool, tt.args)
			if got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}

			if strings.Contains(got, vendorBody) {
				t.Errorf("message = %q holds the vendor body", got)
			}
		})
	}
}
