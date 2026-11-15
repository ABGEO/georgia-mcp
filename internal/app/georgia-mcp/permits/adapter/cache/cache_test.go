package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingSource counts the calls of each method, fails with an outage once failing is
// set, and has no application once missing is set.
type countingSource struct {
	match, application atomic.Int32
	failing, missing   atomic.Bool
}

func (c *countingSource) Match(context.Context, app.SearchFilter) (app.MatchRead, error) {
	c.match.Add(1)

	return app.MatchRead{Total: 1, ReadAt: time.Now()}, c.fail()
}

func (c *countingSource) Application(
	_ context.Context,
	number domain.ApplicationNumber,
) (app.ApplicationRead, error) {
	c.application.Add(1)

	if c.missing.Load() {
		return app.ApplicationRead{}, app.ErrApplicationNotFound
	}

	read := app.ApplicationRead{Application: domain.Application{Number: number}, ReadAt: time.Now()}

	return read, c.fail()
}

func (c *countingSource) fail() error {
	if c.failing.Load() {
		return fmt.Errorf("calling TAS: %w", kernel.ErrSourceUnavailable)
	}

	return nil
}

func newCache(t *testing.T, source app.ApplicationSource) *cache.SourceCache {
	t.Helper()

	sources, err := cache.NewSourceCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = sources.Close() })

	return sources
}

func filter(t *testing.T) app.SearchFilter {
	t.Helper()

	code, err := domain.ParseCadastralCode("01.18.04.020.005")
	if err != nil {
		t.Fatal(err)
	}

	from, err := kernel.ParseDate("2026-11-04")
	if err != nil {
		t.Fatal(err)
	}

	period, err := domain.NewPeriod(from, from.AddDays(6))
	if err != nil {
		t.Fatal(err)
	}

	return app.SearchFilter{Code: code, Period: period}
}

func number(t *testing.T) domain.ApplicationNumber {
	t.Helper()

	parsed, err := domain.ParseApplicationNumber("AR11168188")
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// readBoth reads each method once and returns the stale flags.
func readBoth(t *testing.T, sources *cache.SourceCache) (bool, bool, error) {
	t.Helper()

	match, matchErr := sources.Match(t.Context(), filter(t))
	application, applicationErr := sources.Application(t.Context(), number(t))

	return match.Stale, application.Stale, errors.Join(matchErr, applicationErr)
}

// TASPERM-SC-018: A search and an application stay for 1 hour.
func TestSourceCacheTimesToLive(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		sources := newCache(t, source)

		steps := []struct {
			sleep time.Duration
			want  int32
		}{
			{0, 1},
			{59 * time.Minute, 1},
			{time.Minute, 2},
		}

		for _, step := range steps {
			time.Sleep(step.sleep)

			if _, _, err := readBoth(t, sources); err != nil {
				t.Fatal(err)
			}

			if source.match.Load() != step.want || source.application.Load() != step.want {
				t.Fatalf("after %v: calls = %d, %d, want %d", step.sleep, source.match.Load(),
					source.application.Load(), step.want)
			}
		}
	})
}

// TASPERM-SC-018: Another period of the same code is another entry.
func TestSourceCacheKeys(t *testing.T) {
	t.Parallel()

	source := &countingSource{}
	sources := newCache(t, source)

	other := filter(t)
	other.Period = domain.Period{}

	for _, asked := range []app.SearchFilter{filter(t), filter(t), other} {
		if _, err := sources.Match(t.Context(), asked); err != nil {
			t.Fatal(err)
		}
	}

	if source.match.Load() != 2 {
		t.Errorf("calls = %d, want 2", source.match.Load())
	}
}

// TASPERM-SC-019: While TAS fails, a value stays stale until it is 24 hours old.
func TestSourceCacheStale(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		sources := newCache(t, source)

		if _, _, err := readBoth(t, sources); err != nil {
			t.Fatal(err)
		}

		source.failing.Store(true)
		time.Sleep(23*time.Hour + 59*time.Minute)

		matchStale, applicationStale, err := readBoth(t, sources)
		if err != nil || !matchStale || !applicationStale {
			t.Errorf("at 23 hours 59 minutes = %t, %t, %v, want both stale", matchStale,
				applicationStale, err)
		}

		time.Sleep(time.Minute + time.Second)

		_, _, err = readBoth(t, sources)
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("at 24 hours error = %v, want ErrSourceUnavailable", err)
		}
	})
}

// TASPERM-SC-018: A missing application is not kept.
func TestSourceCacheKeepsNoMissingApplication(t *testing.T) {
	t.Parallel()

	source := &countingSource{}
	source.missing.Store(true)
	sources := newCache(t, source)

	for range 2 {
		if _, err := sources.Application(
			t.Context(),
			number(t),
		); !errors.Is(
			err,
			app.ErrApplicationNotFound,
		) {
			t.Fatalf("Application() error = %v, want ErrApplicationNotFound", err)
		}
	}

	if source.application.Load() != 2 {
		t.Errorf("calls = %d, want 2", source.application.Load())
	}
}
