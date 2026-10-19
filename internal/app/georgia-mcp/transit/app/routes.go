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

// RouteSearch asks the routes that match a text, or every route with no text.
type RouteSearch struct {
	Text string
	// Kind is empty for any kind.
	Kind     domain.Kind
	Language kernel.Language
	Limit    int
}

// RouteMatches is the routes of a search, best match first.
type RouteMatches struct {
	Routes []domain.Route
	ReadAt time.Time
	Stale  bool
}

// SearchRoutes finds the routes whose number or name matches a text.
type SearchRoutes struct {
	network NetworkSource
}

// NewSearchRoutes returns the use case.
func NewSearchRoutes(network NetworkSource) *SearchRoutes {
	return &SearchRoutes{network: network}
}

// Search returns the matching routes, up to the limit. With no text, it returns every
// route by kind, then by number.
func (u *SearchRoutes) Search(ctx context.Context, query RouteSearch) (RouteMatches, error) {
	list, err := u.network.Routes(ctx)
	if err != nil {
		return RouteMatches{}, fmt.Errorf("searching the routes: %w", err)
	}

	text := normalize(query.Text)
	kinds := domain.RouteKinds()

	var (
		matches []ranked[domain.Route]
		all     []domain.Route
	)

	for _, route := range list.Routes {
		if query.Kind != "" && route.Kind != query.Kind {
			continue
		}

		if text == "" {
			all = append(all, route)

			continue
		}

		rank := rankName(route.Name, text)
		if normalize(route.Number) == text {
			rank = rankEqual
		}

		if rank != noMatch {
			matches = append(matches, ranked[domain.Route]{item: route, rank: rank})
		}
	}

	routes := all
	if text == "" {
		slices.SortStableFunc(routes, func(a, b domain.Route) int {
			return cmp.Or(
				cmp.Compare(slices.Index(kinds, a.Kind), slices.Index(kinds, b.Kind)),
				compareNumbers(a.Number, b.Number),
			)
		})
	} else {
		routes = sortRanked(
			matches,
			func(r domain.Route) domain.Name { return r.Name },
			func(r domain.Route) string { return string(r.ID) },
			query.Language,
		)
	}

	return RouteMatches{
		Routes: limit(routes, query.Limit),
		ReadAt: list.ReadAt,
		Stale:  list.Stale,
	}, nil
}

// RouteQuery asks one route.
type RouteQuery struct {
	// Route is the identifier or the number of the route.
	Route string
	// Kind chooses between routes that share a number. Empty accepts any kind.
	Kind domain.Kind
}

// DirectionView is a direction with its stops.
type DirectionView struct {
	ID       int
	Headsign domain.Name
	Stops    []domain.Stop
}

// RouteView is a route with its directions.
type RouteView struct {
	Route      domain.Route
	Directions []DirectionView
	ReadAt     time.Time
	Stale      bool
}

// GetRoute finds a route and its directions.
type GetRoute struct {
	network NetworkSource
	routes  RouteSource
}

// NewGetRoute returns the use case.
func NewGetRoute(network NetworkSource, routes RouteSource) *GetRoute {
	return &GetRoute{network: network, routes: routes}
}

// Get returns the route with each of its directions and their stops in order.
func (u *GetRoute) Get(ctx context.Context, query RouteQuery) (RouteView, error) {
	routes, err := u.network.Routes(ctx)
	if err != nil {
		return RouteView{}, fmt.Errorf("reading the routes: %w", err)
	}

	route, err := resolveRoute(routes.Routes, query.Route, query.Kind)
	if err != nil {
		return RouteView{}, err
	}

	stops, err := u.network.Stops(ctx)
	if err != nil {
		return RouteView{}, fmt.Errorf("reading the stops: %w", err)
	}

	directions, err := u.routes.Directions(ctx, route.ID)
	if err != nil {
		return RouteView{}, fmt.Errorf("reading the directions of %s: %w", route.ID, err)
	}

	byID := make(map[domain.StopID]domain.Stop, len(stops.Stops))
	for _, stop := range stops.Stops {
		byID[stop.ID] = stop
	}

	views := make([]DirectionView, 0, len(directions.Directions))

	for _, direction := range directions.Directions {
		view := DirectionView{ID: direction.ID, Headsign: direction.Headsign}

		for _, id := range direction.Stops {
			stop, found := byID[id]
			if !found {
				stop = domain.Stop{ID: id}
			}

			view.Stops = append(view.Stops, stop)
		}

		views = append(views, view)
	}

	return RouteView{
		Route:      route,
		Directions: views,
		ReadAt:     directions.ReadAt,
		Stale:      directions.Stale,
	}, nil
}
