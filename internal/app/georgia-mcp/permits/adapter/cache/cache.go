// Package cache keeps the applications in memory, in front of the port of the context.
package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	// A decision changes a status on its day. A value is stale for at most 24 hours in all.
	ttl                = time.Hour
	maxAge             = 24 * time.Hour
	matchEntries       = 200
	applicationEntries = 1000
)

// SourceCache implements app.ApplicationSource in front of another source.
type SourceCache struct {
	source       app.ApplicationSource
	matches      *cache.Cache[app.MatchRead]
	applications *cache.Cache[app.ApplicationRead]
}

var (
	_ app.ApplicationSource = (*SourceCache)(nil)
	_ io.Closer             = (*SourceCache)(nil)
)

// NewSourceCache returns a cache in front of the source.
func NewSourceCache(source app.ApplicationSource) (*SourceCache, error) {
	matches, matchesErr := cache.New[app.MatchRead](cache.Config{
		TTL: ttl, MaxStale: maxAge - ttl, MaxEntries: matchEntries,
	})
	applications, applicationsErr := cache.New[app.ApplicationRead](cache.Config{
		TTL: ttl, MaxStale: maxAge - ttl, MaxEntries: applicationEntries,
	})

	if err := errors.Join(matchesErr, applicationsErr); err != nil {
		for _, built := range []interface{ Close() }{matches, applications} {
			if built != nil {
				built.Close()
			}
		}

		return nil, fmt.Errorf("building the stores of the applications: %w", err)
	}

	return &SourceCache{source: source, matches: matches, applications: applications}, nil
}

// Match returns the cached matches of the filter, or reads them.
func (c *SourceCache) Match(ctx context.Context, filter app.SearchFilter) (app.MatchRead, error) {
	key := filter.Code.String() + "|" + date(filter.Period.From) + "|" + date(filter.Period.To)

	result, err := c.matches.Get(ctx, key, func(ctx context.Context) (app.MatchRead, error) {
		return c.source.Match(ctx, filter)
	})
	if err != nil {
		return app.MatchRead{}, fmt.Errorf("searching the applications: %w", err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// Application returns the cached application, or reads it. A missing application is not
// kept.
func (c *SourceCache) Application(
	ctx context.Context,
	number domain.ApplicationNumber,
) (app.ApplicationRead, error) {
	result, err := c.applications.Get(
		ctx,
		number.String(),
		func(ctx context.Context) (app.ApplicationRead, error) {
			return c.source.Application(ctx, number)
		},
	)
	if err != nil {
		return app.ApplicationRead{}, fmt.Errorf("reading the application: %w", err)
	}

	read := result.Value
	read.Stale = result.Stale

	return read, nil
}

// Close stops the goroutines of the stores.
func (c *SourceCache) Close() error {
	c.matches.Close()
	c.applications.Close()

	return nil
}

func date(day kernel.Date) string {
	if day.IsZero() {
		return ""
	}

	return day.String()
}
