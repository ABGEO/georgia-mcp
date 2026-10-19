package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingTTC answers an empty read at the current instant, counts each call, and fails
// with an outage once failing is set.
type countingTTC struct {
	stops      atomic.Int32
	routes     atomic.Int32
	timetables atomic.Int32
	arrivals   atomic.Int32
	failing    atomic.Bool
}

func (c *countingTTC) Stops(context.Context) (app.StopList, error) {
	c.stops.Add(1)

	return app.StopList{ReadAt: time.Now()}, c.outage()
}

func (c *countingTTC) Routes(context.Context) (app.RouteList, error) {
	c.routes.Add(1)

	return app.RouteList{ReadAt: time.Now()}, c.outage()
}

func (c *countingTTC) Directions(context.Context, domain.RouteID) (app.DirectionList, error) {
	return app.DirectionList{ReadAt: time.Now()}, c.outage()
}

func (c *countingTTC) Timetable(context.Context, domain.RouteID, int) (app.TimetableRead, error) {
	c.timetables.Add(1)

	return app.TimetableRead{ReadAt: time.Now()}, c.outage()
}

func (c *countingTTC) Arrivals(context.Context, app.ArrivalRequest) (app.ArrivalList, error) {
	c.arrivals.Add(1)

	return app.ArrivalList{ReadAt: time.Now()}, c.outage()
}

func (c *countingTTC) outage() error {
	if c.failing.Load() {
		return fmt.Errorf("calling the TTC: %w", kernel.ErrSourceUnavailable)
	}

	return nil
}

// TTCTRANS-SC-040: The stops stay in the cache for 24 hours. After that, a failed read
// returns the old list, marked as stale.
func TestNetworkCacheKeepsTheListsForADay(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingTTC{}

		network, err := cache.NewNetworkCache(source)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _ = network.Close() })

		first, err := network.Stops(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(24*time.Hour - time.Minute)

		if _, err := network.Stops(t.Context()); err != nil || source.stops.Load() != 1 {
			t.Fatalf("Stops() at 23:59 = %v with %d calls, want 1 call", err, source.stops.Load())
		}

		time.Sleep(time.Minute)
		source.failing.Store(true)

		stale, err := network.Stops(t.Context())
		if err != nil || source.stops.Load() != 2 {
			t.Fatalf("Stops() at 24:00 = %v with %d calls, want 2 calls", err, source.stops.Load())
		}

		if !stale.Stale || !stale.ReadAt.Equal(first.ReadAt) {
			t.Errorf(
				"Stops() = stale %v, read %v, want the stale list of %v",
				stale.Stale,
				stale.ReadAt,
				first.ReadAt,
			)
		}
	})
}

// TTCTRANS-SC-040: A timetable read before midnight in Tbilisi is not served after it,
// so a timetable read on Sunday is not served on Monday.
func TestRouteCacheReadsATimetableOnANewDate(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingTTC{}

		routes, err := cache.NewRouteCache(source, time.Now)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _ = routes.Close() })

		// The bubble starts at 00:00 UTC, which is 04:00 in Tbilisi.
		time.Sleep(18 * time.Hour)

		if _, err := routes.Timetable(t.Context(), "1:R29925", 1); err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Hour)

		if _, err := routes.Timetable(
			t.Context(),
			"1:R29925",
			1,
		); err != nil ||
			source.timetables.Load() != 1 {
			t.Fatalf(
				"Timetable() at 23:00 = %v with %d calls, want 1",
				err,
				source.timetables.Load(),
			)
		}

		time.Sleep(90 * time.Minute)

		if _, err := routes.Timetable(
			t.Context(),
			"1:R29925",
			1,
		); err != nil ||
			source.timetables.Load() != 2 {
			t.Errorf(
				"Timetable() at 00:30 = %v with %d calls, want 2",
				err,
				source.timetables.Load(),
			)
		}
	})
}

// TTCTRANS-SC-041: The arrivals stay in the cache for 15 seconds, and a failed read has
// no stale value.
func TestArrivalCacheKeepsTheArrivalsFor15Seconds(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingTTC{}

		arrivals, err := cache.NewArrivalCache(source)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _ = arrivals.Close() })

		request := app.ArrivalRequest{Stop: "1:1000", Language: kernel.LanguageEnglish}

		for _, wait := range []time.Duration{0, 14 * time.Second} {
			time.Sleep(wait)

			if _, err := arrivals.Arrivals(t.Context(), request); err != nil {
				t.Fatal(err)
			}
		}

		if calls := source.arrivals.Load(); calls != 1 {
			t.Fatalf("two reads 14 seconds apart sent %d calls, want 1", calls)
		}

		time.Sleep(2 * time.Second)

		if _, err := arrivals.Arrivals(
			t.Context(),
			request,
		); err != nil ||
			source.arrivals.Load() != 2 {
			t.Fatalf(
				"Arrivals() at 16 seconds = %v with %d calls, want 2",
				err,
				source.arrivals.Load(),
			)
		}

		time.Sleep(16 * time.Second)
		source.failing.Store(true)

		if _, err := arrivals.Arrivals(
			t.Context(),
			request,
		); !errors.Is(
			err,
			kernel.ErrSourceUnavailable,
		) {
			t.Errorf("Arrivals() during an outage error = %v, want ErrSourceUnavailable", err)
		}
	})
}
