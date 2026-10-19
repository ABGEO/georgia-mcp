// Package cache keeps the answers of the TTC in memory, in front of each port of the
// context.
package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// listTTL bounds the age of the stops and the routes. The TTC changes them rarely.
	listTTL = 24 * time.Hour
	// timetableTTL bounds the age of a timetable. The key also holds the date, so a
	// timetable of the week before is never served.
	timetableTTL = 6 * time.Hour
	// arrivalTTL lets two clients that ask one stop at once share one call. The web
	// application of the TTC reads the arrivals each 30 seconds.
	arrivalTTL = 15 * time.Second
	placeTTL   = time.Hour

	// maxListStale keeps the stops and the routes for 48 hours after their read.
	maxListStale = 24 * time.Hour
	// maxAge bounds the age of a stale timetable or a stale place list.
	maxAge = 24 * time.Hour

	listEntries    = 1
	routeEntries   = 512
	arrivalEntries = 1024
	placeEntries   = 512
	stopsKey       = "stops"
	routesKey      = "routes"
	keySeparator   = "|"
)

// NetworkCache implements app.NetworkSource in front of another source.
type NetworkCache struct {
	source app.NetworkSource
	stops  *cache.Cache[app.StopList]
	routes *cache.Cache[app.RouteList]
}

var _ app.NetworkSource = (*NetworkCache)(nil)

// NewNetworkCache returns a cache in front of the source.
func NewNetworkCache(source app.NetworkSource) (*NetworkCache, error) {
	config := cache.Config{TTL: listTTL, MaxStale: maxListStale, MaxEntries: listEntries}

	stops, err := cache.New[app.StopList](config)
	if err != nil {
		return nil, fmt.Errorf("building the store of the stops: %w", err)
	}

	routes, err := cache.New[app.RouteList](config)
	if err != nil {
		stops.Close()

		return nil, fmt.Errorf("building the store of the routes: %w", err)
	}

	return &NetworkCache{source: source, stops: stops, routes: routes}, nil
}

// Stops returns the cached list, or reads it.
func (c *NetworkCache) Stops(ctx context.Context) (app.StopList, error) {
	result, err := c.stops.Get(ctx, stopsKey, c.source.Stops)
	if err != nil {
		return app.StopList{}, fmt.Errorf("reading the stops: %w", err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Routes returns the cached list, or reads it.
func (c *NetworkCache) Routes(ctx context.Context) (app.RouteList, error) {
	result, err := c.routes.Get(ctx, routesKey, c.source.Routes)
	if err != nil {
		return app.RouteList{}, fmt.Errorf("reading the routes: %w", err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Close stops the goroutines of the stores.
func (c *NetworkCache) Close() error {
	c.stops.Close()
	c.routes.Close()

	return nil
}

// RouteCache implements app.RouteSource in front of another source.
type RouteCache struct {
	source     app.RouteSource
	now        func() time.Time
	directions *cache.Cache[app.DirectionList]
	timetables *cache.Cache[app.TimetableRead]
}

var _ app.RouteSource = (*RouteCache)(nil)

// NewRouteCache returns a cache in front of the source. now gives the date of the key
// of a timetable.
func NewRouteCache(source app.RouteSource, now func() time.Time) (*RouteCache, error) {
	directions, err := cache.New[app.DirectionList](cache.Config{
		TTL: listTTL, MaxStale: maxListStale, MaxEntries: routeEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("building the store of the directions: %w", err)
	}

	timetables, err := cache.New[app.TimetableRead](cache.Config{
		TTL: timetableTTL, MaxStale: maxAge - timetableTTL, MaxEntries: routeEntries,
	})
	if err != nil {
		directions.Close()

		return nil, fmt.Errorf("building the store of the timetables: %w", err)
	}

	return &RouteCache{
		source:     source,
		now:        now,
		directions: directions,
		timetables: timetables,
	}, nil
}

// Directions returns the cached directions of the route, or reads them.
func (c *RouteCache) Directions(
	ctx context.Context,
	route domain.RouteID,
) (app.DirectionList, error) {
	result, err := c.directions.Get(
		ctx,
		string(route),
		func(ctx context.Context) (app.DirectionList, error) {
			return c.source.Directions(ctx, route)
		},
	)
	if err != nil {
		return app.DirectionList{}, fmt.Errorf("reading the directions of %s: %w", route, err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Timetable returns the cached timetable of the direction for the date in Tbilisi,
// or reads it.
func (c *RouteCache) Timetable(
	ctx context.Context,
	route domain.RouteID,
	direction int,
) (app.TimetableRead, error) {
	key := join(string(route), strconv.Itoa(direction), kernel.Today(c.now()).String())

	result, err := c.timetables.Get(ctx, key, func(ctx context.Context) (app.TimetableRead, error) {
		return c.source.Timetable(ctx, route, direction)
	})
	if err != nil {
		return app.TimetableRead{}, fmt.Errorf("reading the timetable of %s: %w", route, err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// Close stops the goroutines of the stores.
func (c *RouteCache) Close() error {
	c.directions.Close()
	c.timetables.Close()

	return nil
}

// ArrivalCache implements app.ArrivalSource in front of another source. It keeps no
// stale value: an old arrival is a wrong arrival.
type ArrivalCache struct {
	source   app.ArrivalSource
	arrivals *cache.Cache[app.ArrivalList]
}

var _ app.ArrivalSource = (*ArrivalCache)(nil)

// NewArrivalCache returns a cache in front of the source.
func NewArrivalCache(source app.ArrivalSource) (*ArrivalCache, error) {
	arrivals, err := cache.New[app.ArrivalList](cache.Config{
		TTL: arrivalTTL, MaxEntries: arrivalEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("building the store of the arrivals: %w", err)
	}

	return &ArrivalCache{source: source, arrivals: arrivals}, nil
}

// Arrivals returns the cached arrivals of the request, or reads them.
func (c *ArrivalCache) Arrivals(
	ctx context.Context,
	request app.ArrivalRequest,
) (app.ArrivalList, error) {
	key := join(
		string(request.Stop),
		string(request.Route),
		strconv.Itoa(request.Max),
		string(request.Language),
	)

	result, err := c.arrivals.Get(ctx, key, func(ctx context.Context) (app.ArrivalList, error) {
		return c.source.Arrivals(ctx, request)
	})
	if err != nil {
		return app.ArrivalList{}, fmt.Errorf("reading the arrivals at %s: %w", request.Stop, err)
	}

	return result.Value, nil
}

// Close stops the goroutines of the store.
func (c *ArrivalCache) Close() error {
	c.arrivals.Close()

	return nil
}

// PlaceCache implements app.PlaceFinder in front of another source.
type PlaceCache struct {
	source app.PlaceFinder
	places *cache.Cache[app.PlaceList]
}

var _ app.PlaceFinder = (*PlaceCache)(nil)

// NewPlaceCache returns a cache in front of the source.
func NewPlaceCache(source app.PlaceFinder) (*PlaceCache, error) {
	places, err := cache.New[app.PlaceList](cache.Config{
		TTL: placeTTL, MaxStale: maxAge - placeTTL, MaxEntries: placeEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("building the store of the places: %w", err)
	}

	return &PlaceCache{source: source, places: places}, nil
}

// Places returns the cached places of the text, or reads them.
func (c *PlaceCache) Places(
	ctx context.Context,
	text string,
	language kernel.Language,
) (app.PlaceList, error) {
	key := join(strings.ToLower(text), string(language))

	result, err := c.places.Get(ctx, key, func(ctx context.Context) (app.PlaceList, error) {
		return c.source.Places(ctx, text, language)
	})
	if err != nil {
		return app.PlaceList{}, fmt.Errorf("searching the places: %w", err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Close stops the goroutines of the store.
func (c *PlaceCache) Close() error {
	c.places.Close()

	return nil
}

func join(parts ...string) string {
	return strings.Join(parts, keySeparator)
}
