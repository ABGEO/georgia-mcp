package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const day = 24 * time.Hour

// countingNEA answers an empty read at the current instant, counts each call, and fails
// with an outage once failing is set.
type countingNEA struct {
	calls   atomic.Int32
	failing atomic.Bool
}

func (c *countingNEA) Current(context.Context) (app.Network, error) {
	if err := c.answer(); err != nil {
		return app.Network{}, err
	}

	return app.Network{ReadAt: time.Now()}, nil
}

func (c *countingNEA) Recent(context.Context, domain.StationCode) (app.Network, error) {
	if err := c.answer(); err != nil {
		return app.Network{}, err
	}

	return app.Network{ReadAt: time.Now()}, nil
}

func (c *countingNEA) History(
	context.Context,
	domain.StationCode,
	time.Time,
	time.Time,
) (app.Network, error) {
	if err := c.answer(); err != nil {
		return app.Network{}, err
	}

	return app.Network{ReadAt: time.Now()}, nil
}

func (c *countingNEA) Cities(context.Context) (app.CityList, error) {
	if err := c.answer(); err != nil {
		return app.CityList{}, err
	}

	return app.CityList{ReadAt: time.Now()}, nil
}

func (c *countingNEA) answer() error {
	c.calls.Add(1)

	if c.failing.Load() {
		return fmt.Errorf("calling the NEA: %w", kernel.ErrSourceUnavailable)
	}

	return nil
}

func newReadingCache(t *testing.T, source app.ReadingSource) *cache.ReadingCache {
	t.Helper()

	readings, err := cache.NewReadingCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = readings.Close() })

	return readings
}

func recent(t *testing.T, readings *cache.ReadingCache) (app.Network, error) {
	t.Helper()

	code, err := domain.ParseStationCode("TSRT")
	if err != nil {
		t.Fatal(err)
	}

	return readings.Recent(t.Context(), code)
}

func history(t *testing.T, readings *cache.ReadingCache) (app.Network, error) {
	t.Helper()

	code, err := domain.ParseStationCode("TSRT")
	if err != nil {
		t.Fatal(err)
	}

	from := time.Date(2026, time.November, 1, 0, 0, 0, 0, kernel.TbilisiZone())

	return readings.History(t.Context(), code, from, from.Add(95*time.Hour))
}

// NEAAIR-SC-020: The current readings stay in the cache for 15 minutes.
func TestReadingCacheKeepsCurrentFor15Minutes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingNEA{}
		readings := newReadingCache(t, source)

		if _, err := readings.Current(t.Context()); err != nil {
			t.Fatal(err)
		}

		time.Sleep(15*time.Minute - time.Second)

		if _, err := readings.Current(t.Context()); err != nil || source.calls.Load() != 1 {
			t.Fatalf("Current() at 14:59 = %v with %d calls, want 1 call", err, source.calls.Load())
		}

		time.Sleep(time.Second)

		if _, err := readings.Current(t.Context()); err != nil || source.calls.Load() != 2 {
			t.Fatalf(
				"Current() at 15:00 = %v with %d calls, want 2 calls",
				err,
				source.calls.Load(),
			)
		}
	})
}

// NEAAIR-SC-021: While the NEA fails, the cache returns the readings as stale until they
// are 3 hours old, for each method.
func TestReadingCacheKeepsStaleReadingsFor3Hours(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingNEA{}
		readings := newReadingCache(t, source)

		methods := map[string]func() (app.Network, error){
			"Current": func() (app.Network, error) { return readings.Current(t.Context()) },
			"Recent":  func() (app.Network, error) { return recent(t, readings) },
			"History": func() (app.Network, error) { return history(t, readings) },
		}

		for name, read := range methods {
			if _, err := read(); err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}
		}

		source.failing.Store(true)
		time.Sleep(2 * time.Hour)

		for name, read := range methods {
			if network, err := read(); err != nil || !network.Stale {
				t.Errorf("%s() at 2 hours = stale %t, %v, want stale", name, network.Stale, err)
			}
		}

		time.Sleep(time.Hour + time.Second)

		for name, read := range methods {
			if _, err := read(); !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("%s() at 3 hours and 1 second error = %v, want ErrSourceUnavailable",
					name, err)
			}
		}
	})
}

// NEAAIR-SC-022: The cities stay for 24 hours, and as stale until they are 7 days old.
func TestCityCache(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingNEA{}

		cities, err := cache.NewCityCache(source)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _ = cities.Close() })

		if _, err := cities.Cities(t.Context()); err != nil {
			t.Fatal(err)
		}

		time.Sleep(day - time.Minute)

		if _, err := cities.Cities(t.Context()); err != nil || source.calls.Load() != 1 {
			t.Fatalf("Cities() at 23:59 = %v with %d calls, want 1 call", err, source.calls.Load())
		}

		source.failing.Store(true)
		time.Sleep(5*day + time.Minute)

		stale, err := cities.Cities(t.Context())
		if err != nil || !stale.Stale {
			t.Fatalf("Cities() at 6 days = stale %t, %v, want stale", stale.Stale, err)
		}

		time.Sleep(day + time.Second)

		if _, err := cities.Cities(t.Context()); !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("Cities() at 7 days and 1 second error = %v, want ErrSourceUnavailable", err)
		}
	})
}
