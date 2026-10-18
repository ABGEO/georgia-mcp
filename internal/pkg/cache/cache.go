// Package cache keeps the values of a source in memory, for a time to live and a
// maximum stale age.
package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maypok86/otter/v2"
	"golang.org/x/sync/singleflight"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

var (
	// ErrInvalidConfig tells that a cache configuration is not valid.
	ErrInvalidConfig = errors.New("cache: invalid configuration")

	errUnexpectedValue = errors.New("cache: unexpected value type")
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
	store    *otter.Cache[string, entry[V]]
	group    singleflight.Group
	ttl      time.Duration
	maxStale time.Duration
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

// The shared load ignores the cancel of the first caller, so the other callers
// still get the value. The timeout of the vendor client still ends it.
func (c *Cache[V]) load(
	ctx context.Context,
	key string,
	load func(context.Context) (V, error),
) (entry[V], error) {
	shared := c.group.DoChan(key, func() (any, error) {
		value, err := load(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}

		loaded := entry[V]{value: value, fetchedAt: time.Now()}
		c.store.Set(key, loaded)

		return loaded, nil
	})

	select {
	case <-ctx.Done():
		return entry[V]{}, fmt.Errorf("waiting for the cache: %w", ctx.Err())
	case result := <-shared:
		if result.Err != nil {
			return entry[V]{}, result.Err
		}

		loaded, ok := result.Val.(entry[V])
		if !ok {
			return entry[V]{}, fmt.Errorf("%w: %T", errUnexpectedValue, result.Val)
		}

		return loaded, nil
	}
}
