// Package cache keeps the grid reads of GSE and the books of GNERC in memory, in front of
// the ports of the context.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	// gridTTL bounds the age of a grid read. GSE adds a slot each 3 minutes.
	gridTTL = time.Minute
	// gridMaxAge bounds the age of a stale grid read.
	gridMaxAge = 30 * time.Minute

	slotEntries = 1024
	newestKey   = "newest"
)

// GridCache implements app.GridSource in front of another source.
type GridCache struct {
	source app.GridSource
	newest *cache.Cache[app.GridRead]
	slots  *cache.Cache[app.GridRead]
}

var _ app.GridSource = (*GridCache)(nil)

// NewGridCache returns a cache in front of the source.
func NewGridCache(source app.GridSource) (*GridCache, error) {
	newest, err := newStore(1)
	if err != nil {
		return nil, err
	}

	slots, err := newStore(slotEntries)
	if err != nil {
		newest.Close()

		return nil, err
	}

	return &GridCache{source: source, newest: newest, slots: slots}, nil
}

func newStore(entries int) (*cache.Cache[app.GridRead], error) {
	store, err := cache.New[app.GridRead](cache.Config{
		TTL: gridTTL, MaxStale: gridMaxAge - gridTTL, MaxEntries: entries,
	})
	if err != nil {
		return nil, fmt.Errorf("building a store of GSE: %w", err)
	}

	return store, nil
}

// Newest returns the cached newest grid read, or reads it. The key holds across the
// slots, so a stale read outlives the turn of a slot.
func (c *GridCache) Newest(ctx context.Context) (app.GridRead, error) {
	result, err := c.newest.Get(ctx, newestKey, c.source.Newest)
	if err != nil {
		return app.GridRead{}, fmt.Errorf("reading the newest grid: %w", err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// At returns the cached grid read of the slot, or reads it.
func (c *GridCache) At(ctx context.Context, slot time.Time) (app.GridRead, error) {
	key := slot.UTC().Format(time.RFC3339)

	result, err := c.slots.Get(ctx, key, func(ctx context.Context) (app.GridRead, error) {
		return c.source.At(ctx, slot)
	})
	if err != nil {
		return app.GridRead{}, fmt.Errorf("reading the grid of %s: %w", key, err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// Close stops the goroutines of the stores.
func (c *GridCache) Close() error {
	c.newest.Close()
	c.slots.Close()

	return nil
}
