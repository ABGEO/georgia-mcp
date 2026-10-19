// Package cache keeps the price list of each fuel company in memory, in front of its port.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	// ttl bounds the delay of a new price. A company changes its prices at most one
	// time in a day.
	ttl = 30 * time.Minute
	// maxStale keeps a list for 12 hours after its read, with the time to live.
	maxStale = 12*time.Hour - ttl
	// key is the one entry of a company: one call returns every product.
	key = "prices"
)

// PriceCache implements app.PriceSource in front of another source.
type PriceCache struct {
	source app.PriceSource
	lists  *cache.Cache[app.PriceList]
}

var _ app.PriceSource = (*PriceCache)(nil)

// New returns a cache in front of the source.
func New(source app.PriceSource) (*PriceCache, error) {
	lists, err := cache.New[app.PriceList](
		cache.Config{TTL: ttl, MaxStale: maxStale, MaxEntries: 1},
	)
	if err != nil {
		return nil, fmt.Errorf("building the store of the prices: %w", err)
	}

	return &PriceCache{source: source, lists: lists}, nil
}

// Company returns the company of the source.
func (c *PriceCache) Company() domain.Company {
	return c.source.Company()
}

// Prices returns the cached list, or reads it. ReadAt is the read instant that the
// source gave.
func (c *PriceCache) Prices(ctx context.Context) (app.PriceList, error) {
	result, err := c.lists.Get(ctx, key, c.source.Prices)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the prices of %s: %w",
			c.source.Company().Name,
			err,
		)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Close stops the goroutines of the store.
func (c *PriceCache) Close() error {
	c.lists.Close()

	return nil
}
