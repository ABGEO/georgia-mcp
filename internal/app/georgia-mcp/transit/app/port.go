package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// StopList is every stop, the instant the server read it, and whether it is a list
// read before an outage of the TTC.
type StopList struct {
	Stops  []domain.Stop
	ReadAt time.Time
	Stale  bool
}

// RouteList is every route, with the facts of its read.
type RouteList struct {
	Routes []domain.Route
	ReadAt time.Time
	Stale  bool
}

// DirectionList is the directions of one route, with the facts of its read.
type DirectionList struct {
	Directions []domain.Direction
	ReadAt     time.Time
	Stale      bool
}

// TimetableRead is the timetable of one direction, with the facts of its read.
type TimetableRead struct {
	Timetable domain.Timetable
	ReadAt    time.Time
	Stale     bool
}

// ArrivalList is the next arrivals at a stop and the instant the server read them.
type ArrivalList struct {
	Arrivals []domain.Arrival
	ReadAt   time.Time
}

// TripPlan is the itineraries of a trip and the instant the server read them.
type TripPlan struct {
	Itineraries []domain.Itinerary
	ReadAt      time.Time
}

// PlaceList is the places that match a text, with the facts of its read.
type PlaceList struct {
	Places []domain.Place
	ReadAt time.Time
	Stale  bool
}

// ArrivalRequest asks the arrivals at one stop.
type ArrivalRequest struct {
	Stop domain.StopID
	// Route is empty for the next arrival of each route.
	Route domain.RouteID
	// Max is 0 for the default of the source.
	Max      int
	Language kernel.Language
}

// TripRequest asks the itineraries between two points.
type TripRequest struct {
	From   domain.Point
	To     domain.Point
	Timing domain.Timing
	// At is zero with TimingLeaveNow.
	At         time.Time
	Kinds      []domain.Kind
	Preference domain.Preference
	Language   kernel.Language
}

// NetworkSource reads the stops and the routes of the TTC, with the names in Georgian
// and in English.
type NetworkSource interface {
	Stops(ctx context.Context) (StopList, error)
	Routes(ctx context.Context) (RouteList, error)
}

// RouteSource reads the directions and the timetable of a route.
type RouteSource interface {
	Directions(ctx context.Context, route domain.RouteID) (DirectionList, error)
	// Timetable returns ErrUnknownDirection when the route has no such direction.
	Timetable(ctx context.Context, route domain.RouteID, direction int) (TimetableRead, error)
}

// ArrivalSource reads the next arrivals at a stop.
type ArrivalSource interface {
	Arrivals(ctx context.Context, request ArrivalRequest) (ArrivalList, error)
}

// TripPlanner plans a trip between two points.
type TripPlanner interface {
	// Plan returns ErrTripNotFound when the source cannot plan the trip.
	Plan(ctx context.Context, request TripRequest) (TripPlan, error)
}

// PlaceFinder finds the places in Tbilisi that match a text.
type PlaceFinder interface {
	Places(ctx context.Context, text string, language kernel.Language) (PlaceList, error)
}
