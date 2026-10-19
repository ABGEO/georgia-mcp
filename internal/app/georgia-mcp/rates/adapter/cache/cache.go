// Package cache keeps the official rate sets and the commercial rates in memory, in
// front of their ports.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// currentTTL bounds the delay of a new set after about 17:00.
	currentTTL = 15 * time.Minute
	// pastTTL is long, because a set of the past never changes.
	pastTTL  = 30 * 24 * time.Hour
	maxStale = 7 * 24 * time.Hour
)

// RateSetCache implements app.OfficialRateSource in front of another source. It keeps
// the sets of today and later apart from the sets of the past.
type RateSetCache struct {
	source  app.OfficialRateSource
	current *cache.Cache[domain.RateSet]
	past    *cache.Cache[domain.RateSet]
	now     func() time.Time
}

var _ app.OfficialRateSource = (*RateSetCache)(nil)

// New returns a cache in front of the source. Each store holds at most maxEntries sets.
func New(
	source app.OfficialRateSource,
	maxEntries int,
	now func() time.Time,
) (*RateSetCache, error) {
	current, err := cache.New[domain.RateSet](
		cache.Config{TTL: currentTTL, MaxStale: maxStale, MaxEntries: maxEntries},
	)
	if err != nil {
		return nil, fmt.Errorf("building the store of current sets: %w", err)
	}

	past, err := cache.New[domain.RateSet](
		cache.Config{TTL: pastTTL, MaxStale: maxStale, MaxEntries: maxEntries},
	)
	if err != nil {
		current.Close()

		return nil, fmt.Errorf("building the store of past sets: %w", err)
	}

	return &RateSetCache{source: source, current: current, past: past, now: now}, nil
}

// RateSet returns the cached set of the date and the language, or reads it.
func (c *RateSetCache) RateSet(
	ctx context.Context,
	date kernel.Date,
	language kernel.Language,
) (app.RateSetResult, error) {
	store := c.current
	if date.Before(kernel.Today(c.now())) {
		store = c.past
	}

	key := date.String() + "/" + string(language)

	result, err := store.Get(ctx, key, func(ctx context.Context) (domain.RateSet, error) {
		read, err := c.source.RateSet(ctx, date, language)

		return read.Set, err
	})
	if err != nil {
		return app.RateSetResult{}, err
	}

	return app.RateSetResult{Set: result.Value, ReadAt: result.FetchedAt, Stale: result.Stale}, nil
}

// Close stops the goroutines of both stores.
func (c *RateSetCache) Close() error {
	c.current.Close()
	c.past.Close()

	return nil
}
