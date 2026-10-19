package tool

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// StopSearcher finds stops by text.
type StopSearcher interface {
	Search(ctx context.Context, query app.StopSearch) (app.StopMatches, error)
}

// NearbyStopLister finds the stops near a point.
type NearbyStopLister interface {
	List(ctx context.Context, query app.NearbyQuery) (app.NearbyStops, error)
}

// ArrivalLister finds the next arrivals at a stop.
type ArrivalLister interface {
	List(ctx context.Context, query app.ArrivalQuery) (app.Arrivals, error)
}

// RouteSearcher finds routes by text.
type RouteSearcher interface {
	Search(ctx context.Context, query app.RouteSearch) (app.RouteMatches, error)
}

// RouteGetter finds a route and its directions.
type RouteGetter interface {
	Get(ctx context.Context, query app.RouteQuery) (app.RouteView, error)
}

// TimetableGetter finds the times of a stop on a direction.
type TimetableGetter interface {
	Get(ctx context.Context, query app.TimetableQuery) (app.TimetableView, error)
}

// Planner plans a trip.
type Planner interface {
	Plan(ctx context.Context, query app.TripQuery) (app.Trip, error)
}

// PlaceSearcher finds places by text.
type PlaceSearcher interface {
	Search(ctx context.Context, query app.PlaceQuery) (app.Places, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	SearchStops  StopSearcher
	NearbyStops  NearbyStopLister
	Arrivals     ArrivalLister
	SearchRoutes RouteSearcher
	Route        RouteGetter
	Timetable    TimetableGetter
	Trip         Planner
	Places       PlaceSearcher
}

const (
	source = "TTC"

	maxQuery        = 100
	maxRadius       = 2000
	defaultStops    = 10
	maxStops        = 50
	defaultArrivals = 10
	maxArrivals     = 30
	defaultRoutes   = 20
	maxRoutes       = 100
	defaultPlaces   = 5
	maxPlaces       = 20
	defaultRadius   = 500
	minutesInDay    = 24 * 60
)

const (
	searchStopsDescription = "Finds the stops of the Tbilisi Transport Company (bus stops, metro " +
		"stations, cable car stations) by name or stop code, in Georgian or English. Use the stop " +
		"id of the result with transit_list_arrivals or transit_get_timetable."
	nearbyStopsDescription = "Lists the stops of the TTC within a distance of a point in " +
		"Tbilisi, nearest first, with the distance in metres. Use transit_search_places to get " +
		"the coordinates of a place."
	arrivalsDescription = "Lists the next vehicles of the TTC at a stop: route number, kind, " +
		"headsign, minutes from as_of, and the expected instant. live is true when the TTC " +
		"tracks the vehicle; else the time comes from the timetable. With a route, lists the " +
		"next vehicles of that route only."
	searchRoutesDescription = "Finds the routes of the TTC by number or by name. Kinds: bus, " +
		"minibus, metro, cable_car. With no query, lists every route."
	routeDescription = "Returns a route of the TTC with each direction: its headsign and its " +
		"stops in order."
	timetableDescription = "Returns the planned departure times of one direction of a route " +
		"of the TTC at one stop, on a date of the current week. A time of 24:00 or later is " +
		"after midnight, on the next date."
	planTripDescription = "Plans a trip by public transport of the TTC in Tbilisi between two " +
		"points, given as coordinates. Returns up to 3 itineraries with each leg: walk or ride, " +
		"route, stops, instants, distance in metres. Use transit_search_places to get coordinates."
	searchPlacesDescription = "Finds places in Tbilisi (addresses, streets, buildings) by text, " +
		"with their coordinates, for transit_plan_trip and transit_list_nearby_stops. Data from " +
		"OpenStreetMap through the TTC."
)

// Register adds the eight tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := errors.Join(
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_search_stops", Title: "Search the stops of Tbilisi",
			Description: searchStopsDescription,
		}, handlers.searchStops),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_list_nearby_stops", Title: "Stops near a point in Tbilisi",
			Description: nearbyStopsDescription,
		}, handlers.nearbyStops),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_list_arrivals", Title: "Next arrivals at a stop in Tbilisi",
			Description: arrivalsDescription,
		}, handlers.arrivals),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_search_routes", Title: "Search the routes of Tbilisi",
			Description: searchRoutesDescription,
		}, handlers.searchRoutes),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_get_route", Title: "A route of Tbilisi and its stops",
			Description: routeDescription,
		}, handlers.route),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_get_timetable", Title: "Timetable of a route at a stop",
			Description: timetableDescription,
		}, handlers.timetable),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_plan_trip", Title: "Plan a trip by public transport in Tbilisi",
			Description: planTripDescription,
		}, handlers.planTrip),
		mcpserver.AddTool(registry, &mcp.Tool{
			Name: "transit_search_places", Title: "Search places in Tbilisi",
			Description: searchPlacesDescription,
		}, handlers.searchPlaces),
	)
	if err != nil {
		return fmt.Errorf("registering the transit tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) searchStops(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchStopsInput,
) (*mcp.CallToolResult, stopsOutput, error) {
	language, kind, count, err := commonStopInput(in.Language, in.Kind, in.Limit)
	if err != nil {
		return nil, stopsOutput{}, err
	}

	if err := checkQuery(in.Query, true); err != nil {
		return nil, stopsOutput{}, err
	}

	matches, err := h.uses.SearchStops.Search(ctx, app.StopSearch{
		Text: in.Query, Kind: kind, Language: language, Limit: count,
	})
	if err != nil {
		return nil, stopsOutput{}, toolError(err)
	}

	return nil, stopsOutput{
		Source: source,
		AsOf:   instant(matches.ReadAt),
		Stale:  matches.Stale,
		Stops:  toStops(matches.Stops, language),
	}, nil
}

func (h handler) nearbyStops(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in nearbyStopsInput,
) (*mcp.CallToolResult, nearbyOutput, error) {
	language, kind, count, err := commonStopInput(in.Language, in.Kind, in.Limit)
	if err != nil {
		return nil, nearbyOutput{}, err
	}

	radius, err := withinRange("radius_m", in.RadiusM, defaultRadius, maxRadius)
	if err != nil {
		return nil, nearbyOutput{}, err
	}

	point, err := toPoint("lat, lon", in.Lat, in.Lon)
	if err != nil {
		return nil, nearbyOutput{}, err
	}

	nearby, err := h.uses.NearbyStops.List(ctx, app.NearbyQuery{
		Point: point, Radius: float64(radius), Kind: kind, Limit: count,
	})
	if err != nil {
		return nil, nearbyOutput{}, toolError(err)
	}

	stops := make([]nearbyStopOutput, 0, len(nearby.Stops))
	for _, stop := range nearby.Stops {
		stops = append(
			stops,
			nearbyStopOutput{Stop: toStop(stop.Stop, language), DistanceM: stop.Metres},
		)
	}

	return nil, nearbyOutput{
		Source: source, AsOf: instant(nearby.ReadAt), Stale: nearby.Stale, Stops: stops,
	}, nil
}

func (h handler) arrivals(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listArrivalsInput,
) (*mcp.CallToolResult, arrivalsOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, arrivalsOutput{}, err
	}

	count, err := withinRange("limit", in.Limit, defaultArrivals, maxArrivals)
	if err != nil {
		return nil, arrivalsOutput{}, err
	}

	result, err := h.uses.Arrivals.List(ctx, app.ArrivalQuery{
		Stop: in.Stop, Route: in.Route, Language: language, Limit: count,
	})
	if err != nil {
		return nil, arrivalsOutput{}, toolError(err)
	}

	return nil, toArrivals(result, language), nil
}

func (h handler) searchRoutes(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchRoutesInput,
) (*mcp.CallToolResult, routesOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, routesOutput{}, err
	}

	kind, err := optionalRouteKind(in.Kind)
	if err != nil {
		return nil, routesOutput{}, err
	}

	count, err := withinRange("limit", in.Limit, defaultRoutes, maxRoutes)
	if err != nil {
		return nil, routesOutput{}, err
	}

	if err := checkQuery(in.Query, false); err != nil {
		return nil, routesOutput{}, err
	}

	matches, err := h.uses.SearchRoutes.Search(ctx, app.RouteSearch{
		Text: in.Query, Kind: kind, Language: language, Limit: count,
	})
	if err != nil {
		return nil, routesOutput{}, toolError(err)
	}

	routes := make([]routeOutput, 0, len(matches.Routes))
	for _, route := range matches.Routes {
		routes = append(routes, toRoute(route, language))
	}

	return nil, routesOutput{
		Source: source, AsOf: instant(matches.ReadAt), Stale: matches.Stale, Routes: routes,
	}, nil
}

func (h handler) route(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getRouteInput,
) (*mcp.CallToolResult, routeDetailOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, routeDetailOutput{}, err
	}

	kind, err := optionalRouteKind(in.Kind)
	if err != nil {
		return nil, routeDetailOutput{}, err
	}

	view, err := h.uses.Route.Get(ctx, app.RouteQuery{Route: in.Route, Kind: kind})
	if err != nil {
		return nil, routeDetailOutput{}, toolError(err)
	}

	return nil, toRouteDetail(view, language), nil
}

func (h handler) timetable(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getTimetableInput,
) (*mcp.CallToolResult, timetableOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, timetableOutput{}, err
	}

	kind, err := optionalRouteKind(in.Kind)
	if err != nil {
		return nil, timetableOutput{}, err
	}

	date, err := optionalDate(in.Date)
	if err != nil {
		return nil, timetableOutput{}, err
	}

	view, err := h.uses.Timetable.Get(ctx, app.TimetableQuery{
		Route: in.Route, Kind: kind, Direction: in.Direction, Stop: in.Stop, Date: date,
	})
	if err != nil {
		return nil, timetableOutput{}, toolError(err)
	}

	return nil, toTimetable(view, language), nil
}

func (h handler) planTrip(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in planTripInput,
) (*mcp.CallToolResult, tripOutput, error) {
	query, err := toTripQuery(in)
	if err != nil {
		return nil, tripOutput{}, err
	}

	trip, err := h.uses.Trip.Plan(ctx, query)
	if err != nil {
		return nil, tripOutput{}, toolError(err)
	}

	return nil, toTrip(trip), nil
}

func (h handler) searchPlaces(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchPlacesInput,
) (*mcp.CallToolResult, placesOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, placesOutput{}, err
	}

	count, err := withinRange("limit", in.Limit, defaultPlaces, maxPlaces)
	if err != nil {
		return nil, placesOutput{}, err
	}

	if err := checkQuery(in.Query, true); err != nil {
		return nil, placesOutput{}, err
	}

	places, err := h.uses.Places.Search(
		ctx,
		app.PlaceQuery{Text: in.Query, Language: language, Limit: count},
	)
	if err != nil {
		return nil, placesOutput{}, toolError(err)
	}

	return nil, toPlaces(places), nil
}

func commonStopInput(
	languageText, kindText string,
	limit int,
) (kernel.Language, domain.Kind, int, error) {
	language, err := parseLanguage(languageText)
	if err != nil {
		return "", "", 0, err
	}

	var kind domain.Kind

	if kindText != "" {
		kind, err = domain.ParseStopKind(kindText)
		if err != nil {
			return "", "", 0, invalidKind("kind", "stop", kindText, domain.StopKinds(), err)
		}
	}

	count, err := withinRange("limit", limit, defaultStops, maxStops)
	if err != nil {
		return "", "", 0, err
	}

	return language, kind, count, nil
}

func toTripQuery(in planTripInput) (app.TripQuery, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return app.TripQuery{}, err
	}

	from, err := toPoint("from", in.From.Lat, in.From.Lon)
	if err != nil {
		return app.TripQuery{}, err
	}

	to, err := toPoint("to", in.To.Lat, in.To.Lon)
	if err != nil {
		return app.TripQuery{}, err
	}

	query, err := tripOptions(in)
	if err != nil {
		return app.TripQuery{}, err
	}

	query.From, query.To, query.Language = from, to, language

	return query, nil
}

// tripOptions reads the timing, the kinds, and the preference of a trip.
func tripOptions(in planTripInput) (app.TripQuery, error) {
	timing, err := domain.ParseTiming(in.Timing)
	if err != nil {
		return app.TripQuery{}, invalidTiming(in.Timing, err)
	}

	clock, err := tripTime(timing, in.Time)
	if err != nil {
		return app.TripQuery{}, err
	}

	date, err := optionalDate(in.Date)
	if err != nil {
		return app.TripQuery{}, err
	}

	for _, text := range in.Kinds {
		if _, err := domain.ParseRouteKind(text); err != nil {
			return app.TripQuery{}, invalidKind("kinds", "route", text, domain.RouteKinds(), err)
		}
	}

	kinds, err := domain.ParseTripKinds(in.Kinds)
	if err != nil {
		return app.TripQuery{}, invalidTripKinds(err)
	}

	preference, err := domain.ParsePreference(in.Preference)
	if err != nil {
		return app.TripQuery{}, invalidPreference(in.Preference, err)
	}

	return app.TripQuery{
		Timing: timing, Date: date, Time: clock, Kinds: kinds, Preference: preference,
	}, nil
}

// tripTime returns the time of depart_at or arrive_by: HH:MM of a day, below 24:00.
func tripTime(timing domain.Timing, text string) (*domain.ClockTime, error) {
	if timing == domain.TimingLeaveNow {
		return nil, nil //nolint:nilnil // Leave now has no time, and no error.
	}

	if text == "" {
		return nil, missingTime(timing)
	}

	clock, err := domain.ParseClockTime(text)
	if err != nil || clock.Minutes() >= minutesInDay {
		return nil, invalidTime(text, err)
	}

	return &clock, nil
}
