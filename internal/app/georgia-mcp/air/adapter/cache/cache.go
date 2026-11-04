// Package cache keeps the readings and the cities of the NEA in memory, in front of the
// ports of the context.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	// readingTTL bounds the age of a reading. The NEA adds one reading each hour.
	readingTTL = 15 * time.Minute
	// readingMaxAge bounds the age of a stale reading.
	readingMaxAge = 3 * time.Hour
	// cityTTL bounds the age of the cities. They change rarely.
	cityTTL = 24 * time.Hour
	// cityMaxAge bounds the age of stale cities.
	cityMaxAge = 7 * 24 * time.Hour

	historyEntries = 256
	currentKey     = "current"
	citiesKey      = "cities"
	keySeparator   = "|"
)

func newStore[V any](ttl, maxAge time.Duration, entries int) (*cache.Cache[V], error) {
	store, err := cache.New[V](cache.Config{TTL: ttl, MaxStale: maxAge - ttl, MaxEntries: entries})
	if err != nil {
		return nil, fmt.Errorf("building a store of the NEA: %w", err)
	}

	return store, nil
}

// ReadingCache implements app.ReadingSource in front of another source.
type ReadingCache struct {
	source  app.ReadingSource
	current *cache.Cache[app.Network]
	recent  *cache.Cache[app.Network]
	history *cache.Cache[app.Network]
}

var _ app.ReadingSource = (*ReadingCache)(nil)

// NewReadingCache returns a cache in front of the source.
func NewReadingCache(source app.ReadingSource) (*ReadingCache, error) {
	current, err := newStore[app.Network](readingTTL, readingMaxAge, 1)
	if err != nil {
		return nil, err
	}

	recent, err := newStore[app.Network](readingTTL, readingMaxAge, historyEntries)
	if err != nil {
		current.Close()

		return nil, err
	}

	history, err := newStore[app.Network](readingTTL, readingMaxAge, historyEntries)
	if err != nil {
		current.Close()
		recent.Close()

		return nil, err
	}

	return &ReadingCache{source: source, current: current, recent: recent, history: history}, nil
}

// Current returns the cached readings of every station, or reads them.
func (c *ReadingCache) Current(ctx context.Context) (app.Network, error) {
	result, err := c.current.Get(ctx, currentKey, c.source.Current)
	if err != nil {
		return app.Network{}, fmt.Errorf("reading the current readings: %w", err)
	}

	network := result.Value
	network.Stale = result.Stale

	return network, nil
}

// Recent returns the cached recent readings of the station, or reads them. The key is the
// code alone, so a stale value outlives the turn of the hour.
func (c *ReadingCache) Recent(ctx context.Context, code domain.StationCode) (app.Network, error) {
	result, err := c.recent.Get(ctx, code.String(), func(ctx context.Context) (app.Network, error) {
		return c.source.Recent(ctx, code)
	})
	if err != nil {
		return app.Network{}, fmt.Errorf("reading the recent readings: %w", err)
	}

	network := result.Value
	network.Stale = result.Stale

	return network, nil
}

// History returns the cached readings of the station between the two hours, or reads them.
func (c *ReadingCache) History(
	ctx context.Context,
	code domain.StationCode,
	from, to time.Time,
) (app.Network, error) {
	key := code.String() + keySeparator + from.UTC().Format(time.RFC3339) + keySeparator +
		to.UTC().Format(time.RFC3339)

	result, err := c.history.Get(ctx, key, func(ctx context.Context) (app.Network, error) {
		return c.source.History(ctx, code, from, to)
	})
	if err != nil {
		return app.Network{}, fmt.Errorf("reading the history: %w", err)
	}

	network := result.Value
	network.Stale = result.Stale

	return network, nil
}

// Close stops the goroutines of the stores.
func (c *ReadingCache) Close() error {
	c.current.Close()
	c.recent.Close()
	c.history.Close()

	return nil
}

// CityCache implements app.CitySource in front of another source.
type CityCache struct {
	source app.CitySource
	cities *cache.Cache[app.CityList]
}

var _ app.CitySource = (*CityCache)(nil)

// NewCityCache returns a cache in front of the source.
func NewCityCache(source app.CitySource) (*CityCache, error) {
	cities, err := newStore[app.CityList](cityTTL, cityMaxAge, 1)
	if err != nil {
		return nil, err
	}

	return &CityCache{source: source, cities: cities}, nil
}

// Cities returns the cached cities, or reads them.
func (c *CityCache) Cities(ctx context.Context) (app.CityList, error) {
	result, err := c.cities.Get(ctx, citiesKey, c.source.Cities)
	if err != nil {
		return app.CityList{}, fmt.Errorf("reading the cities: %w", err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Close stops the goroutines of the store.
func (c *CityCache) Close() error {
	c.cities.Close()

	return nil
}
