package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	day  = 24 * time.Hour
	year = 2026
)

// countingSource answers an empty calendar at the current instant, counts each call, and
// fails with an outage once failing is set.
type countingSource struct {
	calls   atomic.Int32
	failing atomic.Bool
}

func (c *countingSource) Holidays(_ context.Context, year int) (app.Calendar, error) {
	c.calls.Add(1)

	if c.failing.Load() {
		return app.Calendar{}, fmt.Errorf("calling Nager.Date: %w", kernel.ErrSourceUnavailable)
	}

	return app.Calendar{Year: year, ReadAt: time.Now()}, nil
}

func newCache(t *testing.T, source app.HolidaySource) *cache.CalendarCache {
	t.Helper()

	calendars, err := cache.NewCalendarCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = calendars.Close() })

	return calendars
}

// HOLIDAY-SC-016: A calendar stays in the cache for 7 days.
func TestCalendarCacheKeepsAYearFor7Days(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		calendars := newCache(t, source)

		if _, err := calendars.Holidays(t.Context(), year); err != nil {
			t.Fatal(err)
		}

		time.Sleep(7*day - time.Minute)

		if _, err := calendars.Holidays(t.Context(), year); err != nil || source.calls.Load() != 1 {
			t.Fatalf(
				"Holidays() at 6 days 23:59 = %v with %d calls, want 1 call",
				err,
				source.calls.Load(),
			)
		}

		time.Sleep(time.Minute)

		if _, err := calendars.Holidays(t.Context(), year); err != nil || source.calls.Load() != 2 {
			t.Fatalf(
				"Holidays() at 7 days = %v with %d calls, want 2 calls",
				err,
				source.calls.Load(),
			)
		}
	})
}

// HOLIDAY-SC-017: While the source fails, the cache returns the calendar as stale until it
// is 30 days old.
func TestCalendarCacheKeepsAStaleYearFor30Days(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		calendars := newCache(t, source)

		if _, err := calendars.Holidays(t.Context(), year); err != nil {
			t.Fatal(err)
		}

		source.failing.Store(true)
		time.Sleep(29 * day)

		calendar, err := calendars.Holidays(t.Context(), year)
		if err != nil || !calendar.Stale || calendar.Year != year {
			t.Errorf(
				"Holidays() at 29 days = stale %t, %v, want the stale calendar",
				calendar.Stale,
				err,
			)
		}

		time.Sleep(day + time.Second)

		if _, err := calendars.Holidays(
			t.Context(),
			year,
		); !errors.Is(
			err,
			kernel.ErrSourceUnavailable,
		) {
			t.Errorf(
				"Holidays() at 30 days and 1 second error = %v, want ErrSourceUnavailable",
				err,
			)
		}
	})
}
