package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// ArrivalQuery asks the next arrivals at a stop.
type ArrivalQuery struct {
	// Stop is the identifier or the code of the stop.
	Stop string
	// Route is the identifier or the number of a route. Empty asks every route.
	Route    string
	Language kernel.Language
	Limit    int
}

// Arrivals is the next arrivals at a stop, soonest first.
type Arrivals struct {
	Stop domain.Stop
	// Route is nil when the query names no route.
	Route    *domain.Route
	Arrivals []domain.Arrival
	ReadAt   time.Time
}

// ListArrivals finds the next arrivals at a stop.
type ListArrivals struct {
	network  NetworkSource
	arrivals ArrivalSource
}

// NewListArrivals returns the use case.
func NewListArrivals(network NetworkSource, arrivals ArrivalSource) *ListArrivals {
	return &ListArrivals{network: network, arrivals: arrivals}
}

// List returns the next arrivals, up to the limit.
func (u *ListArrivals) List(ctx context.Context, query ArrivalQuery) (Arrivals, error) {
	stops, err := u.network.Stops(ctx)
	if err != nil {
		return Arrivals{}, fmt.Errorf("reading the stops: %w", err)
	}

	stop, err := resolveStop(stops.Stops, query.Stop)
	if err != nil {
		return Arrivals{}, err
	}

	routes, err := u.network.Routes(ctx)
	if err != nil {
		return Arrivals{}, fmt.Errorf("reading the routes: %w", err)
	}

	request := ArrivalRequest{Stop: stop.ID, Language: query.Language}

	var route *domain.Route

	if query.Route != "" {
		atStop := slices.DeleteFunc(slices.Clone(routes.Routes), func(r domain.Route) bool {
			return r.Kind.StopKind() != stop.Kind
		})

		found, err := resolveRoute(atStop, query.Route, "")
		if err != nil {
			return Arrivals{}, err
		}

		route = &found
		request.Route = found.ID
		request.Max = query.Limit
	}

	list, err := u.arrivals.Arrivals(ctx, request)
	if err != nil {
		return Arrivals{}, fmt.Errorf("reading the arrivals at %s: %w", stop.ID, err)
	}

	arrivals := slices.Clone(list.Arrivals)
	for i := range arrivals {
		arrivals[i].Kind = classify(routes.Routes, arrivals[i].RouteNumber, stop.Kind)
	}

	slices.SortStableFunc(arrivals, func(a, b domain.Arrival) int {
		return cmp.Or(
			cmp.Compare(a.Minutes, b.Minutes),
			compareNumbers(a.RouteNumber, b.RouteNumber),
		)
	})

	return Arrivals{
		Stop:     stop,
		Route:    route,
		Arrivals: limit(arrivals, query.Limit),
		ReadAt:   list.ReadAt,
	}, nil
}
