package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// TimetableQuery asks the times of one direction of a route at one stop on one date.
type TimetableQuery struct {
	// Route is the identifier or the number of the route.
	Route string
	// Kind chooses between routes that share a number. Empty accepts any kind.
	Kind domain.Kind
	// Direction is nil for the one direction of a route that has one.
	Direction *int
	// Stop is the identifier or the code of the stop.
	Stop string
	// Date is zero for the date in Tbilisi.
	Date kernel.Date
}

// TimetableView is the times of a stop on a date.
type TimetableView struct {
	Route     domain.Route
	Direction domain.Direction
	Stop      domain.Stop
	Date      kernel.Date
	Times     []domain.ClockTime
	ReadAt    time.Time
	Stale     bool
}

// GetTimetable finds the planned times of a direction of a route at a stop.
type GetTimetable struct {
	network NetworkSource
	routes  RouteSource
	now     func() time.Time
}

// NewGetTimetable returns the use case. now gives the date of a query with no date.
func NewGetTimetable(
	network NetworkSource,
	routes RouteSource,
	now func() time.Time,
) *GetTimetable {
	return &GetTimetable{network: network, routes: routes, now: now}
}

// Get returns the times, in the order of the TTC.
func (u *GetTimetable) Get(ctx context.Context, query TimetableQuery) (TimetableView, error) {
	route, direction, err := u.direction(ctx, query)
	if err != nil {
		return TimetableView{}, err
	}

	stops, err := u.network.Stops(ctx)
	if err != nil {
		return TimetableView{}, fmt.Errorf("reading the stops: %w", err)
	}

	stop, err := resolveStop(stops.Stops, query.Stop)
	if err != nil {
		return TimetableView{}, err
	}

	read, err := u.routes.Timetable(ctx, route.ID, direction.ID)
	if err != nil {
		return TimetableView{}, fmt.Errorf("reading the timetable of %s: %w", route.ID, err)
	}

	date := query.Date
	if date.IsZero() {
		date = kernel.Today(u.now())
	}

	if !read.Timetable.HasDate(date) {
		first, last := read.Timetable.Span()

		return TimetableView{}, &DateOutsideTimetableError{Date: date, First: first, Last: last}
	}

	times, found := read.Timetable.Times(date, stop.ID)
	if !found {
		return TimetableView{}, &StopNotOnDirectionError{
			Stop:      stop,
			Route:     route,
			Direction: direction.ID,
		}
	}

	return TimetableView{
		Route:     route,
		Direction: direction,
		Stop:      stop,
		Date:      date,
		Times:     times,
		ReadAt:    read.ReadAt,
		Stale:     read.Stale,
	}, nil
}

// direction resolves the route of the query and chooses its direction.
func (u *GetTimetable) direction(
	ctx context.Context,
	query TimetableQuery,
) (domain.Route, domain.Direction, error) {
	routes, err := u.network.Routes(ctx)
	if err != nil {
		return domain.Route{}, domain.Direction{}, fmt.Errorf("reading the routes: %w", err)
	}

	route, err := resolveRoute(routes.Routes, query.Route, query.Kind)
	if err != nil {
		return domain.Route{}, domain.Direction{}, err
	}

	directions, err := u.routes.Directions(ctx, route.ID)
	if err != nil {
		return domain.Route{}, domain.Direction{}, fmt.Errorf(
			"reading the directions of %s: %w", route.ID, err,
		)
	}

	direction, err := chooseDirection(route, directions.Directions, query.Direction)
	if err != nil {
		return domain.Route{}, domain.Direction{}, err
	}

	return route, direction, nil
}

func chooseDirection(
	route domain.Route,
	directions []domain.Direction,
	wanted *int,
) (domain.Direction, error) {
	if wanted == nil {
		if len(directions) == 1 {
			return directions[0], nil
		}

		return domain.Direction{}, &DirectionError{Route: route, Directions: directions}
	}

	index := slices.IndexFunc(directions, func(d domain.Direction) bool { return d.ID == *wanted })
	if index < 0 {
		return domain.Direction{}, &DirectionError{Route: route, Directions: directions}
	}

	return directions[index], nil
}
