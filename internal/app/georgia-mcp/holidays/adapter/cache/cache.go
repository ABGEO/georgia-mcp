// Package cache keeps the public holidays of each year in memory, in front of the port of
// the context.
package cache

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	// calendarTTL bounds the age of a calendar. A calendar changes only with the law or
	// the source.
	calendarTTL = 7 * 24 * time.Hour
	// calendarMaxAge bounds the age of a stale calendar.
	calendarMaxAge = 30 * 24 * time.Hour
	// calendarEntries holds the three supported years and one more at the turn of a year.
	calendarEntries = 4
)

// CalendarCache implements app.HolidaySource in front of another source.
type CalendarCache struct {
	source    app.HolidaySource
	calendars *cache.Cache[app.Calendar]
}

var (
	_ app.HolidaySource = (*CalendarCache)(nil)
	_ io.Closer         = (*CalendarCache)(nil)
)

// NewCalendarCache returns a cache in front of the source.
func NewCalendarCache(source app.HolidaySource) (*CalendarCache, error) {
	calendars, err := cache.New[app.Calendar](cache.Config{
		TTL:        calendarTTL,
		MaxStale:   calendarMaxAge - calendarTTL,
		MaxEntries: calendarEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("building the store of the holidays: %w", err)
	}

	return &CalendarCache{source: source, calendars: calendars}, nil
}

// Holidays returns the cached calendar of the year, or reads it.
func (c *CalendarCache) Holidays(ctx context.Context, year int) (app.Calendar, error) {
	result, err := c.calendars.Get(
		ctx,
		strconv.Itoa(year),
		func(ctx context.Context) (app.Calendar, error) {
			return c.source.Holidays(ctx, year)
		},
	)
	if err != nil {
		return app.Calendar{}, fmt.Errorf("reading the holidays: %w", err)
	}

	calendar := result.Value
	calendar.Stale = result.Stale

	return calendar, nil
}

// Close stops the goroutine of the store.
func (c *CalendarCache) Close() error {
	c.calendars.Close()

	return nil
}
