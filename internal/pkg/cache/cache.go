// Package cache keeps the values of a source in memory, for a time to live and a
// maximum stale age.
package cache

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/maypok86/otter/v2"
	"golang.org/x/sync/singleflight"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

var (
	// ErrInvalidConfig tells that a cache configuration is not valid.
	ErrInvalidConfig = errors.New("cache: invalid configuration")

	errUnexpectedValue = errors.New("cache: unexpected value type")
	errMissingValue    = errors.New("cache: the load returned no value for a key")
)

// Config is the configuration of one cache. A context builds it from its own variables.
type Config struct {
	TTL        time.Duration
	MaxStale   time.Duration
	MaxEntries int
}

// Result is a value and the instant that the source gave it.
type Result[V any] struct {
	Value     V
	FetchedAt time.Time
	Stale     bool
}

type entry[V any] struct {
	value     V
	fetchedAt time.Time
}

// Cache keeps the values of one source. A TTL of zero turns the cache off.
type Cache[V any] struct {
	store     *otter.Cache[string, entry[V]]
	group     singleflight.Group
	groupMany singleflight.Group
	ttl       time.Duration
	maxStale  time.Duration
}

// New returns an empty cache.
func New[V any](cfg Config) (*Cache[V], error) {
	if cfg.TTL < 0 || cfg.MaxStale < 0 {
		return nil, fmt.Errorf("%w: a negative duration", ErrInvalidConfig)
	}

	values := &Cache[V]{ttl: cfg.TTL, maxStale: cfg.MaxStale}
	if cfg.TTL == 0 {
		return values, nil
	}

	if cfg.MaxEntries < 1 {
		return nil, fmt.Errorf(
			"%w: the maximum count of entries must be 1 or more",
			ErrInvalidConfig,
		)
	}

	// The store has no expiry of its own: Get decides freshness and the stale age from
	// the fetch time. An expiry in otter starts a goroutine that ticks each second.
	store, err := otter.New(&otter.Options[string, entry[V]]{MaximumSize: cfg.MaxEntries})
	if err != nil {
		return nil, fmt.Errorf("building the cache store: %w", err)
	}

	values.store = store

	return values, nil
}

// Close stops each goroutine of the store.
func (c *Cache[V]) Close() {
	if c.store != nil {
		c.store.StopAllGoroutines()
	}
}

// Get returns the fresh value of the key, or loads it. Concurrent calls for one key
// share one load. When the load fails with kernel.ErrSourceUnavailable, Get returns
// the stale value inside the maximum stale age.
func (c *Cache[V]) Get(
	ctx context.Context,
	key string,
	load func(context.Context) (V, error),
) (Result[V], error) {
	if c.store == nil {
		value, err := load(ctx)
		if err != nil {
			return Result[V]{}, err
		}

		return Result[V]{Value: value, FetchedAt: time.Now()}, nil
	}

	cached, found := c.store.GetIfPresent(key)
	if found && time.Since(cached.fetchedAt) < c.ttl {
		return Result[V]{Value: cached.value, FetchedAt: cached.fetchedAt}, nil
	}

	loaded, err := c.load(ctx, key, load)
	if err == nil {
		return Result[V]{Value: loaded.value, FetchedAt: loaded.fetchedAt}, nil
	}

	if found && errors.Is(err, kernel.ErrSourceUnavailable) &&
		time.Since(cached.fetchedAt) < c.ttl+c.maxStale {
		return Result[V]{Value: cached.value, FetchedAt: cached.fetchedAt, Stale: true}, nil
	}

	return Result[V]{}, err
}

// load runs the load of one key, shared between the concurrent calls for the key.
func (c *Cache[V]) load(
	ctx context.Context,
	key string,
	load func(context.Context) (V, error),
) (entry[V], error) {
	return share(ctx, &c.group, key, func(ctx context.Context) (entry[V], error) {
		value, err := load(ctx)
		if err != nil {
			return entry[V]{}, err
		}

		loaded := entry[V]{value: value, fetchedAt: time.Now()}
		c.store.Set(key, loaded)

		return loaded, nil
	})
}

// GetMany returns the fresh value of each key, and loads the other keys in one call of
// load. Concurrent calls for the same set of keys share one load. load must return a
// value for each key it gets. When the load fails with kernel.ErrSourceUnavailable,
// GetMany returns the stale value of each key it loaded, if every one of them is
// inside the maximum stale age.
func (c *Cache[V]) GetMany(
	ctx context.Context,
	keys []string,
	load func(ctx context.Context, keys []string) (map[string]V, error),
) (map[string]Result[V], error) {
	if c.store == nil {
		loaded, err := loadAll(ctx, keys, load)
		if err != nil {
			return nil, err
		}

		return toResults(loaded, false), nil
	}

	results, held, pending := c.split(keys)
	if len(pending) == 0 {
		return results, nil
	}

	loaded, err := c.loadMany(ctx, pending, load)
	if err != nil {
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			return nil, err
		}

		loaded, err = c.stale(pending, held, err)
		if err != nil {
			return nil, err
		}

		maps.Copy(results, toResults(loaded, true))

		return results, nil
	}

	maps.Copy(results, toResults(loaded, false))

	return results, nil
}

// split returns the fresh keys as results, the expired entries that the store holds,
// and every key that is not fresh.
func (c *Cache[V]) split(keys []string) (map[string]Result[V], map[string]entry[V], []string) {
	results := make(map[string]Result[V], len(keys))
	held := make(map[string]entry[V])

	var pending []string

	for _, key := range keys {
		cached, found := c.store.GetIfPresent(key)
		if found && time.Since(cached.fetchedAt) < c.ttl {
			results[key] = Result[V]{Value: cached.value, FetchedAt: cached.fetchedAt}

			continue
		}

		if found {
			held[key] = cached
		}

		pending = append(pending, key)
	}

	return results, held, pending
}

// stale returns the held entry of each key, or the error of the load when one key has
// no entry inside the maximum stale age.
func (c *Cache[V]) stale(
	keys []string,
	held map[string]entry[V],
	loadErr error,
) (map[string]entry[V], error) {
	entries := make(map[string]entry[V], len(keys))

	for _, key := range keys {
		cached, found := held[key]
		if !found || time.Since(cached.fetchedAt) >= c.ttl+c.maxStale {
			return nil, loadErr
		}

		entries[key] = cached
	}

	return entries, nil
}

func toResults[V any](entries map[string]entry[V], stale bool) map[string]Result[V] {
	results := make(map[string]Result[V], len(entries))
	for key, value := range entries {
		results[key] = Result[V]{Value: value.value, FetchedAt: value.fetchedAt, Stale: stale}
	}

	return results
}

func (c *Cache[V]) loadMany(
	ctx context.Context,
	keys []string,
	load func(ctx context.Context, keys []string) (map[string]V, error),
) (map[string]entry[V], error) {
	sorted := slices.Sorted(slices.Values(keys))

	return share(
		ctx,
		&c.groupMany,
		strings.Join(sorted, ","),
		func(ctx context.Context) (map[string]entry[V], error) {
			loaded, err := loadAll(ctx, sorted, load)
			if err != nil {
				return nil, err
			}

			for key, value := range loaded {
				c.store.Set(key, value)
			}

			return loaded, nil
		},
	)
}

func loadAll[V any](
	ctx context.Context,
	keys []string,
	load func(ctx context.Context, keys []string) (map[string]V, error),
) (map[string]entry[V], error) {
	values, err := load(ctx, keys)
	if err != nil {
		return nil, err
	}

	fetchedAt := time.Now()
	loaded := make(map[string]entry[V], len(keys))

	for _, key := range keys {
		value, found := values[key]
		if !found {
			return nil, fmt.Errorf("%w: %q", errMissingValue, key)
		}

		loaded[key] = entry[V]{value: value, fetchedAt: fetchedAt}
	}

	return loaded, nil
}

// share runs fn once for the concurrent calls of one key. The shared run ignores the
// cancel of the first caller, so the other callers still get the value. The timeout of
// the vendor client still ends it.
func share[T any](
	ctx context.Context,
	group *singleflight.Group,
	key string,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T

	shared := group.DoChan(key, func() (any, error) {
		return fn(context.WithoutCancel(ctx))
	})

	select {
	case <-ctx.Done():
		return zero, fmt.Errorf("waiting for the cache: %w", ctx.Err())
	case result := <-shared:
		if result.Err != nil {
			return zero, result.Err
		}

		value, ok := result.Val.(T)
		if !ok {
			return zero, fmt.Errorf("%w: %T", errUnexpectedValue, result.Val)
		}

		return value, nil
	}
}
