// Package ttc reads the public transport of Tbilisi from the TTC client and converts
// each answer to the domain. It is the anti-corruption layer of the context.
package ttc

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	ttcclient "github.com/abgeo/georgia-mcp/pkg/ttc"
)

const (
	keyVariable   = "GEOMCP_TRANSIT_TTC_API_KEY"
	minibusPrefix = "1:minibus"
	modeWalk      = "WALK"
	modeBus       = "BUS"
	modeSubway    = "SUBWAY"
	modeGondola   = "GONDOLA"
	dateLayout    = "2006-01-02"
	clockLayout   = "15:04"
)

var errMalformed = errors.New("ttc: malformed answer")

// Adapter implements the ports of the context with the TTC client.
type Adapter struct {
	client *ttcclient.Client
	now    func() time.Time
	logger *slog.Logger
}

var (
	_ app.NetworkSource = (*Adapter)(nil)
	_ app.RouteSource   = (*Adapter)(nil)
	_ app.ArrivalSource = (*Adapter)(nil)
	_ app.TripPlanner   = (*Adapter)(nil)
	_ app.PlaceFinder   = (*Adapter)(nil)
)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives an error each time the TTC refuses the key.
func NewAdapter(client *ttcclient.Client, now func() time.Time, logger *slog.Logger) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Stops reads every stop in Georgian and in English, and joins them by identifier.
func (a *Adapter) Stops(ctx context.Context) (app.StopList, error) {
	georgian, err := a.client.Stops(ctx, string(kernel.LanguageGeorgian))
	if err != nil {
		return app.StopList{}, a.fail(ctx, "reading the stops", err)
	}

	english, err := a.client.Stops(ctx, string(kernel.LanguageEnglish))
	if err != nil {
		return app.StopList{}, a.fail(ctx, "reading the stops", err)
	}

	names := make(map[string]string, len(georgian))
	for _, stop := range georgian {
		names[stop.ID] = stop.Name
	}

	stops := make([]domain.Stop, 0, len(english))

	for _, stop := range english {
		converted, err := toStop(stop, names[stop.ID])
		if err != nil {
			return app.StopList{}, fmt.Errorf(
				"reading the stops: %w: %w",
				kernel.ErrSourceUnavailable,
				err,
			)
		}

		stops = append(stops, converted)
	}

	return app.StopList{Stops: stops, ReadAt: a.now()}, nil
}

// Routes reads every route in Georgian and in English, and joins them by identifier.
func (a *Adapter) Routes(ctx context.Context) (app.RouteList, error) {
	georgian, err := a.client.Routes(ctx, string(kernel.LanguageGeorgian))
	if err != nil {
		return app.RouteList{}, a.fail(ctx, "reading the routes", err)
	}

	english, err := a.client.Routes(ctx, string(kernel.LanguageEnglish))
	if err != nil {
		return app.RouteList{}, a.fail(ctx, "reading the routes", err)
	}

	names := make(map[string]string, len(georgian))
	for _, route := range georgian {
		names[route.ID] = route.LongName
	}

	routes := make([]domain.Route, 0, len(english))

	for _, route := range english {
		kind, err := routeKind(route.ID, route.Mode)
		if err != nil {
			return app.RouteList{}, fmt.Errorf(
				"reading the routes: %w: %w",
				kernel.ErrSourceUnavailable,
				err,
			)
		}

		routes = append(routes, domain.Route{
			ID:     domain.RouteID(route.ID),
			Number: route.ShortName,
			Name:   domain.Name{Georgian: names[route.ID], English: route.LongName},
			Kind:   kind,
		})
	}

	return app.RouteList{Routes: routes, ReadAt: a.now()}, nil
}

// Directions reads the patterns of a route in both languages, and the stops of each.
func (a *Adapter) Directions(ctx context.Context, route domain.RouteID) (app.DirectionList, error) {
	action := "reading the directions of " + string(route)

	georgian, err := a.client.Route(ctx, string(route), string(kernel.LanguageGeorgian))
	if err != nil {
		return app.DirectionList{}, a.fail(ctx, action, err)
	}

	english, err := a.client.Route(ctx, string(route), string(kernel.LanguageEnglish))
	if err != nil {
		return app.DirectionList{}, a.fail(ctx, action, err)
	}

	headsigns := make(map[string]string, len(georgian.Patterns))
	for _, pattern := range georgian.Patterns {
		headsigns[pattern.PatternSuffix] = pattern.Headsign
	}

	directions := make([]domain.Direction, 0, len(english.Patterns))

	for _, pattern := range english.Patterns {
		stops, err := a.client.PatternStops(
			ctx,
			string(route),
			pattern.PatternSuffix,
			string(kernel.LanguageEnglish),
		)
		if err != nil {
			return app.DirectionList{}, a.fail(ctx, action, err)
		}

		ids := make([]domain.StopID, 0, len(stops))
		for _, stop := range stops {
			ids = append(ids, domain.StopID(stop.ID))
		}

		directions = append(directions, domain.Direction{
			ID: pattern.DirectionID,
			Headsign: domain.Name{
				Georgian: headsigns[pattern.PatternSuffix],
				English:  pattern.Headsign,
			},
			Stops: ids,
		})
	}

	return app.DirectionList{Directions: directions, ReadAt: a.now()}, nil
}

// Timetable reads the timetable of the pattern of a direction, one day for each date.
func (a *Adapter) Timetable(
	ctx context.Context,
	route domain.RouteID,
	direction int,
) (app.TimetableRead, error) {
	action := "reading the timetable of " + string(route)

	detail, err := a.client.Route(ctx, string(route), string(kernel.LanguageEnglish))
	if err != nil {
		return app.TimetableRead{}, a.fail(ctx, action, err)
	}

	index := slices.IndexFunc(
		detail.Patterns,
		func(p ttcclient.Pattern) bool { return p.DirectionID == direction },
	)
	if index < 0 {
		return app.TimetableRead{}, fmt.Errorf(
			"%s: %w: %d",
			action,
			app.ErrUnknownDirection,
			direction,
		)
	}

	groups, err := a.client.Schedule(
		ctx, string(route), detail.Patterns[index].PatternSuffix, string(kernel.LanguageEnglish),
	)
	if err != nil {
		return app.TimetableRead{}, a.fail(ctx, action, err)
	}

	timetable, err := toTimetable(groups)
	if err != nil {
		return app.TimetableRead{}, fmt.Errorf(
			"%s: %w: %w",
			action,
			kernel.ErrSourceUnavailable,
			err,
		)
	}

	return app.TimetableRead{Timetable: timetable, ReadAt: a.now()}, nil
}

// Arrivals reads the next arrivals at a stop. A live arrival takes the minutes of
// tracking, any other arrival the minutes of the timetable. An arrival with negative
// minutes is a vehicle that left, so the adapter leaves it out.
func (a *Adapter) Arrivals(
	ctx context.Context,
	request app.ArrivalRequest,
) (app.ArrivalList, error) {
	answer, err := a.client.ArrivalTimes(ctx, string(request.Stop), ttcclient.ArrivalTimesQuery{
		Locale:  locale(request.Language),
		RouteID: string(request.Route),
		Max:     request.Max,
	})
	if err != nil {
		return app.ArrivalList{}, a.fail(ctx, "reading the arrivals at "+string(request.Stop), err)
	}

	readAt := a.now()
	arrivals := make([]domain.Arrival, 0, len(answer))

	for _, arrival := range answer {
		minutes := arrival.ScheduledArrivalMinutes
		if arrival.Realtime {
			minutes = arrival.RealtimeArrivalMinutes
		}

		if minutes < 0 {
			continue
		}

		kind, err := stopKind(arrival.VehicleMode)
		if err != nil {
			return app.ArrivalList{}, fmt.Errorf(
				"reading the arrivals: %w: %w",
				kernel.ErrSourceUnavailable,
				err,
			)
		}

		arrivals = append(arrivals, domain.Arrival{
			RouteNumber: arrival.ShortName,
			Kind:        kind,
			Headsign:    arrival.Headsign,
			Minutes:     minutes,
			Live:        arrival.Realtime,
			Expected:    readAt.Add(time.Duration(minutes) * time.Minute),
		})
	}

	return app.ArrivalList{Arrivals: arrivals, ReadAt: readAt}, nil
}

// Plan reads the itineraries of a trip. A 400 of the TTC is a trip that it cannot plan.
func (a *Adapter) Plan(ctx context.Context, request app.TripRequest) (app.TripPlan, error) {
	answer, err := a.client.Plan(ctx, toPlanQuery(request))
	if apiErr, ok := errors.AsType[*ttcclient.APIError](
		err,
	); ok &&
		apiErr.StatusCode == http.StatusBadRequest {
		return app.TripPlan{}, fmt.Errorf("planning the trip: %w: %w", app.ErrTripNotFound, err)
	}

	if err != nil {
		return app.TripPlan{}, a.fail(ctx, "planning the trip", err)
	}

	itineraries := make([]domain.Itinerary, 0, len(answer.Itineraries))

	for _, itinerary := range answer.Itineraries {
		converted, err := toItinerary(itinerary)
		if err != nil {
			return app.TripPlan{}, fmt.Errorf(
				"planning the trip: %w: %w",
				kernel.ErrSourceUnavailable,
				err,
			)
		}

		itineraries = append(itineraries, converted)
	}

	return app.TripPlan{Itineraries: itineraries, ReadAt: a.now()}, nil
}

// Places reads the places that match a text, inside the box of Tbilisi. A place with
// no two coordinates, or outside Tbilisi, is left out.
func (a *Adapter) Places(
	ctx context.Context,
	text string,
	language kernel.Language,
) (app.PlaceList, error) {
	box := domain.TbilisiBox()

	answer, err := a.client.Geocode(ctx, ttcclient.GeocodeQuery{
		Query:  text,
		Locale: locale(language),
		BBox: strings.Join(
			[]string{degrees(box.West), degrees(box.North), degrees(box.East), degrees(box.South)},
			",",
		),
	})
	if err != nil {
		return app.PlaceList{}, a.fail(ctx, "searching the places", err)
	}

	places := make([]domain.Place, 0, len(answer.Features))

	for _, feature := range answer.Features {
		if place, ok := toPlace(feature); ok {
			places = append(places, place)
		}
	}

	return app.PlaceList{Places: places, ReadAt: a.now()}, nil
}

// fail wraps a failure of the client with kernel.ErrSourceUnavailable. A refused key
// also tells the operator which variable holds it.
func (a *Adapter) fail(ctx context.Context, action string, err error) error {
	if apiErr, ok := errors.AsType[*ttcclient.APIError](
		err,
	); ok &&
		apiErr.StatusCode == http.StatusUnauthorized {
		a.logger.ErrorContext(ctx, "the TTC refused the key", slog.String("variable", keyVariable))
	}

	return fmt.Errorf("%s: %w: %w", action, kernel.ErrSourceUnavailable, err)
}

func toStop(stop ttcclient.Stop, georgian string) (domain.Stop, error) {
	kind, err := stopKind(stop.VehicleMode)
	if err != nil {
		return domain.Stop{}, err
	}

	location, err := domain.NewPoint(stop.Lat, stop.Lon)
	if err != nil {
		return domain.Stop{}, fmt.Errorf("%w: stop %s: %w", errMalformed, stop.ID, err)
	}

	converted := domain.Stop{
		ID:       domain.StopID(stop.ID),
		Name:     domain.Name{Georgian: georgian, English: stop.Name},
		Kind:     kind,
		Location: location,
	}
	if stop.Code != nil {
		converted.Code = *stop.Code
	}

	return converted, nil
}

func stopKind(mode string) (domain.Kind, error) {
	switch mode {
	case modeBus:
		return domain.KindBus, nil
	case modeSubway:
		return domain.KindMetro, nil
	case modeGondola:
		return domain.KindCableCar, nil
	default:
		return "", fmt.Errorf("%w: mode %q", errMalformed, mode)
	}
}

func routeKind(id, mode string) (domain.Kind, error) {
	kind, err := stopKind(mode)
	if err == nil && kind == domain.KindBus && strings.HasPrefix(id, minibusPrefix) {
		return domain.KindMinibus, nil
	}

	return kind, err
}

func toTimetable(groups []ttcclient.ScheduleGroup) (domain.Timetable, error) {
	var days []domain.TimetableDay

	for _, group := range groups {
		stops, err := toStopTimes(group.Stops)
		if err != nil {
			return domain.Timetable{}, err
		}

		for _, text := range group.ServiceDates {
			date, err := kernel.ParseDate(text)
			if err != nil {
				return domain.Timetable{}, fmt.Errorf("%w: %w", errMalformed, err)
			}

			days = append(days, domain.TimetableDay{Date: date, Stops: stops})
		}
	}

	slices.SortFunc(days, func(x, y domain.TimetableDay) int {
		return cmp.Compare(x.Date.String(), y.Date.String())
	})

	return domain.Timetable{Days: days}, nil
}

func toStopTimes(stops []ttcclient.ScheduleStop) ([]domain.StopTimes, error) {
	converted := make([]domain.StopTimes, 0, len(stops))

	for _, stop := range stops {
		var times []domain.ClockTime

		for text := range strings.SplitSeq(stop.ArrivalTimes, ",") {
			if text == "" {
				continue
			}

			clock, err := domain.ParseClockTime(text)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", errMalformed, err)
			}

			times = append(times, clock)
		}

		converted = append(converted, domain.StopTimes{
			Stop: domain.StopID(stop.ID), Position: stop.Position, Times: times,
		})
	}

	slices.SortStableFunc(converted, func(x, y domain.StopTimes) int {
		return cmp.Compare(x.Position, y.Position)
	})

	return converted, nil
}

func toPlanQuery(request app.TripRequest) ttcclient.PlanQuery {
	query := ttcclient.PlanQuery{
		FromLat:    request.From.Lat,
		FromLon:    request.From.Lon,
		ToLat:      request.To.Lat,
		ToLon:      request.To.Lon,
		DepartMode: "leaveNow",
		Modes:      modes(request.Kinds),
		Optimize:   optimize(request.Preference),
		Locale:     locale(request.Language),
	}

	switch request.Timing {
	case domain.TimingDepartAt:
		query.DepartMode = "departAt"
	case domain.TimingArriveBy:
		query.DepartMode = "arriveBy"
	case domain.TimingLeaveNow:
		return query
	}

	at := request.At.In(kernel.TbilisiZone())
	query.Date = at.Format(dateLayout)
	query.Time = at.Format(clockLayout)

	return query
}

// modes returns the modes of the TTC for the kinds. The TTC plans the bus and the
// minibus as one mode.
func modes(kinds []domain.Kind) []string {
	modes := []string{modeWalk}

	if slices.Contains(kinds, domain.KindBus) || slices.Contains(kinds, domain.KindMinibus) {
		modes = append(modes, modeBus)
	}

	if slices.Contains(kinds, domain.KindMetro) {
		modes = append(modes, modeSubway)
	}

	if slices.Contains(kinds, domain.KindCableCar) {
		modes = append(modes, modeGondola)
	}

	return modes
}

func optimize(preference domain.Preference) string {
	switch preference {
	case domain.PreferenceLessWalking:
		return "lessWalking"
	case domain.PreferenceFewerTransfers:
		return "fewerTransfers"
	case domain.PreferenceFastest:
		return "quick"
	default:
		return "quick"
	}
}

func toItinerary(itinerary ttcclient.Itinerary) (domain.Itinerary, error) {
	start, err := instant(itinerary.StartTime)
	if err != nil {
		return domain.Itinerary{}, err
	}

	end, err := instant(itinerary.EndTime)
	if err != nil {
		return domain.Itinerary{}, err
	}

	legs := make([]domain.Leg, 0, len(itinerary.Legs))

	for _, leg := range itinerary.Legs {
		converted, err := toLeg(leg)
		if err != nil {
			return domain.Itinerary{}, err
		}

		legs = append(legs, converted)
	}

	return domain.Itinerary{
		Start:      start,
		End:        end,
		WalkMetres: itinerary.WalkDistance,
		Legs:       legs,
	}, nil
}

func toLeg(leg ttcclient.Leg) (domain.Leg, error) {
	start, err := instant(leg.StartTime)
	if err != nil {
		return domain.Leg{}, err
	}

	end, err := instant(leg.EndTime)
	if err != nil {
		return domain.Leg{}, err
	}

	converted := domain.Leg{
		Kind: domain.KindWalk,
		From: domain.LegEnd{
			Name:     leg.From.Name,
			Location: domain.Point{Lat: leg.From.Lat, Lon: leg.From.Lon},
		},
		To: domain.LegEnd{
			Name:     leg.To.Name,
			Location: domain.Point{Lat: leg.To.Lat, Lon: leg.To.Lon},
		},
		Start:  start,
		End:    end,
		Metres: leg.Distance,
	}

	if leg.Mode == modeWalk {
		return converted, nil
	}

	converted.Kind, err = stopKind(leg.Mode)
	if err != nil {
		return domain.Leg{}, err
	}

	if leg.Route != nil {
		converted.RouteNumber = leg.Route.ShortName
		converted.RouteName = leg.Route.LongName
	}

	converted.Stops = len(leg.IntermediateStops) + 1

	return converted, nil
}

func toPlace(feature ttcclient.Feature) (domain.Place, bool) {
	const coordinates = 2

	if len(feature.Geometry.Coordinates) != coordinates {
		return domain.Place{}, false
	}

	location, err := domain.NewPoint(
		feature.Geometry.Coordinates[1],
		feature.Geometry.Coordinates[0],
	)
	if err != nil || !location.InTbilisi() {
		return domain.Place{}, false
	}

	properties := feature.Properties
	place := domain.Place{
		Name:        properties.Name,
		Street:      properties.Street,
		HouseNumber: properties.HouseNumber,
		District:    cmp.Or(properties.District, properties.Locality),
		Location:    location,
	}

	if place.Name == "" {
		place.Name = strings.TrimSpace(properties.Street + " " + properties.HouseNumber)
	}

	return place, true
}

func instant(text string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return parsed.In(kernel.TbilisiZone()), nil
}

func locale(language kernel.Language) string {
	if language == "" {
		return string(kernel.LanguageEnglish)
	}

	return string(language)
}

func degrees(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
