package app

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

var (
	// ErrEmptyQuery tells that a search has no text.
	ErrEmptyQuery = errors.New("query: empty")
	// ErrUnknownStop tells that a text is neither the identifier nor the code of a stop.
	ErrUnknownStop = errors.New("stop: unknown")
	// ErrUnknownRoute tells that a text is neither the identifier nor the number of a route.
	ErrUnknownRoute = errors.New("route: unknown")
	// ErrAmbiguousRoute tells that a number names two routes or more.
	ErrAmbiguousRoute = errors.New("route: ambiguous number")
	// ErrUnknownDirection tells that a route has no such direction, or needs one.
	ErrUnknownDirection = errors.New("direction: unknown")
	// ErrStopNotOnDirection tells that a stop is not on a direction of a route.
	ErrStopNotOnDirection = errors.New("stop: not on the direction")
	// ErrDateOutsideTimetable tells that a date is outside the published week.
	ErrDateOutsideTimetable = errors.New("date: outside the timetable")
	// ErrOutsideTbilisi tells that a point is outside the box that the TTC serves.
	ErrOutsideTbilisi = errors.New("point: outside Tbilisi")
	// ErrTripNotFound tells that the TTC cannot plan a trip between the two points.
	ErrTripNotFound = errors.New("trip: not found")
)

// UnknownStopError is a text that names no stop.
type UnknownStopError struct {
	Text string
}

func (e *UnknownStopError) Error() string { return fmt.Sprintf("%s: %q", ErrUnknownStop, e.Text) }

// Unwrap returns ErrUnknownStop.
func (e *UnknownStopError) Unwrap() error { return ErrUnknownStop }

// UnknownRouteError is a text that names no route.
type UnknownRouteError struct {
	Text string
}

func (e *UnknownRouteError) Error() string { return fmt.Sprintf("%s: %q", ErrUnknownRoute, e.Text) }

// Unwrap returns ErrUnknownRoute.
func (e *UnknownRouteError) Unwrap() error { return ErrUnknownRoute }

// AmbiguousRouteError is a number of two routes or more.
type AmbiguousRouteError struct {
	Text   string
	Routes []domain.Route
}

func (e *AmbiguousRouteError) Error() string {
	return fmt.Sprintf("%s: %q names %d routes", ErrAmbiguousRoute, e.Text, len(e.Routes))
}

// Unwrap returns ErrAmbiguousRoute.
func (e *AmbiguousRouteError) Unwrap() error { return ErrAmbiguousRoute }

// DirectionError is a direction that the route lacks, or a route of two directions
// with none chosen.
type DirectionError struct {
	Route      domain.Route
	Directions []domain.Direction
}

func (e *DirectionError) Error() string {
	return fmt.Sprintf(
		"%s: route %s has %d directions",
		ErrUnknownDirection,
		e.Route.Number,
		len(e.Directions),
	)
}

// Unwrap returns ErrUnknownDirection.
func (e *DirectionError) Unwrap() error { return ErrUnknownDirection }

// StopNotOnDirectionError is a stop that a direction of a route does not serve.
type StopNotOnDirectionError struct {
	Stop      domain.Stop
	Route     domain.Route
	Direction int
}

func (e *StopNotOnDirectionError) Error() string {
	return fmt.Sprintf(
		"%s: %s, route %s, direction %d",
		ErrStopNotOnDirection,
		e.Stop.ID,
		e.Route.Number,
		e.Direction,
	)
}

// Unwrap returns ErrStopNotOnDirection.
func (e *StopNotOnDirectionError) Unwrap() error { return ErrStopNotOnDirection }

// DateOutsideTimetableError is a date outside the week of a timetable.
type DateOutsideTimetableError struct {
	Date  kernel.Date
	First kernel.Date
	Last  kernel.Date
}

func (e *DateOutsideTimetableError) Error() string {
	return fmt.Sprintf(
		"%s: %s is not from %s to %s",
		ErrDateOutsideTimetable,
		e.Date,
		e.First,
		e.Last,
	)
}

// Unwrap returns ErrDateOutsideTimetable.
func (e *DateOutsideTimetableError) Unwrap() error { return ErrDateOutsideTimetable }

// OutsideTbilisiError is a point of a query outside Tbilisi. Field names the input:
// point, from, or to.
type OutsideTbilisiError struct {
	Field string
	Point domain.Point
}

func (e *OutsideTbilisiError) Error() string {
	return fmt.Sprintf("%s: %s %v, %v", ErrOutsideTbilisi, e.Field, e.Point.Lat, e.Point.Lon)
}

// Unwrap returns ErrOutsideTbilisi.
func (e *OutsideTbilisiError) Unwrap() error { return ErrOutsideTbilisi }
