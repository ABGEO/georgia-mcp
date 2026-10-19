package tool

import (
	"fmt"
	"math"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	noArrival   = "No vehicle is expected at this stop now."
	noItinerary = "The TTC returned no itinerary for this time."
	noPlace     = "No place in Tbilisi matches the text."
)

func toStop(stop domain.Stop, language kernel.Language) stopOutput {
	return stopOutput{
		ID:   string(stop.ID),
		Code: stop.Code,
		Name: stop.Name.In(language),
		Kind: string(stop.Kind),
		Lat:  stop.Location.Lat,
		Lon:  stop.Location.Lon,
	}
}

func toStops(stops []domain.Stop, language kernel.Language) []stopOutput {
	out := make([]stopOutput, 0, len(stops))
	for _, stop := range stops {
		out = append(out, toStop(stop, language))
	}

	return out
}

func toRoute(route domain.Route, language kernel.Language) routeOutput {
	return routeOutput{
		ID:     string(route.ID),
		Number: route.Number,
		Name:   route.Name.In(language),
		Kind:   string(route.Kind),
	}
}

func toArrivals(result app.Arrivals, language kernel.Language) arrivalsOutput {
	out := arrivalsOutput{
		Source:   source,
		AsOf:     instant(result.ReadAt),
		Stop:     toStop(result.Stop, language),
		Arrivals: make([]arrivalOutput, 0, len(result.Arrivals)),
	}

	for _, arrival := range result.Arrivals {
		out.Arrivals = append(out.Arrivals, arrivalOutput{
			Route:    arrival.RouteNumber,
			Kind:     string(arrival.Kind),
			Headsign: arrival.Headsign,
			Minutes:  arrival.Minutes,
			Expected: instant(arrival.Expected),
			Live:     arrival.Live,
		})
	}

	if len(out.Arrivals) == 0 {
		out.Note = noArrival
		if result.Route != nil {
			out.Note = fmt.Sprintf(
				"No vehicle of route %s is expected at this stop now.",
				result.Route.Number,
			)
		}
	}

	return out
}

func toRouteDetail(view app.RouteView, language kernel.Language) routeDetailOutput {
	out := routeDetailOutput{
		Source:     source,
		AsOf:       instant(view.ReadAt),
		Stale:      view.Stale,
		Route:      toRoute(view.Route, language),
		Directions: make([]directionOutput, 0, len(view.Directions)),
	}

	for _, direction := range view.Directions {
		out.Directions = append(out.Directions, directionOutput{
			Direction: direction.ID,
			Headsign:  direction.Headsign.In(language),
			Stops:     toStops(direction.Stops, language),
		})
	}

	return out
}

func toTimetable(view app.TimetableView, language kernel.Language) timetableOutput {
	times := make([]string, 0, len(view.Times))
	for _, clock := range view.Times {
		times = append(times, clock.String())
	}

	return timetableOutput{
		Source:    source,
		AsOf:      instant(view.ReadAt),
		Stale:     view.Stale,
		Route:     toRoute(view.Route, language),
		Direction: view.Direction.ID,
		Headsign:  view.Direction.Headsign.In(language),
		Stop:      toStop(view.Stop, language),
		Date:      view.Date.String(),
		Times:     times,
	}
}

func toTrip(trip app.Trip) tripOutput {
	out := tripOutput{
		Source:      source,
		AsOf:        instant(trip.ReadAt),
		Itineraries: make([]itineraryOutput, 0, len(trip.Itineraries)),
	}

	for _, itinerary := range trip.Itineraries {
		legs := make([]legOutput, 0, len(itinerary.Legs))
		for _, leg := range itinerary.Legs {
			legs = append(legs, toLeg(leg))
		}

		out.Itineraries = append(out.Itineraries, itineraryOutput{
			Start:           instant(itinerary.Start),
			End:             instant(itinerary.End),
			DurationMinutes: int(math.Ceil(itinerary.Duration().Minutes())),
			WalkM:           int(math.Round(itinerary.WalkMetres)),
			Transfers:       itinerary.Transfers(),
			Legs:            legs,
		})
	}

	if len(out.Itineraries) == 0 {
		out.Note = noItinerary
	}

	return out
}

func toLeg(leg domain.Leg) legOutput {
	out := legOutput{
		Kind:      string(leg.Kind),
		From:      toLegPoint(leg.From),
		To:        toLegPoint(leg.To),
		Start:     instant(leg.Start),
		End:       instant(leg.End),
		DistanceM: int(math.Round(leg.Metres)),
	}

	if leg.Kind != domain.KindWalk {
		out.RouteNumber = leg.RouteNumber
		out.RouteName = leg.RouteName
		out.Stops = leg.Stops
	}

	return out
}

func toLegPoint(end domain.LegEnd) legPointOutput {
	return legPointOutput{
		Name:   end.Name,
		StopID: string(end.Stop),
		Lat:    end.Location.Lat,
		Lon:    end.Location.Lon,
	}
}

func toPlaces(places app.Places) placesOutput {
	out := placesOutput{
		Source: source + ", from OpenStreetMap",
		AsOf:   instant(places.ReadAt),
		Stale:  places.Stale,
		Places: make([]placeOutput, 0, len(places.Places)),
	}

	for _, place := range places.Places {
		out.Places = append(out.Places, placeOutput{
			Name:        place.Name,
			Street:      place.Street,
			HouseNumber: place.HouseNumber,
			District:    place.District,
			Lat:         place.Location.Lat,
			Lon:         place.Location.Lon,
		})
	}

	if len(out.Places) == 0 {
		out.Note = noPlace
	}

	return out
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
