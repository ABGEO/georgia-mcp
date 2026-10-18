package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const day = 24 * time.Hour

type countingSource struct {
	calls   atomic.Int32
	failing atomic.Bool
}

func (s *countingSource) RateSet(
	_ context.Context,
	date domain.Date,
	_ kernel.Language,
) (app.RateSetResult, error) {
	s.calls.Add(1)

	if s.failing.Load() {
		return app.RateSetResult{}, fmt.Errorf("calling the NBG: %w", kernel.ErrSourceUnavailable)
	}

	return app.RateSetResult{Set: domain.NewRateSet(date, time.Now(), nil), ReadAt: time.Now()}, nil
}

func newCache(t *testing.T, source app.OfficialRateSource) *cache.RateSetCache {
	t.Helper()

	rateSets, err := cache.New(source, 64, time.Now)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Cleanup(func() { _ = rateSets.Close() })

	return rateSets
}

func ask(
	t *testing.T,
	rateSets *cache.RateSetCache,
	date domain.Date,
	language kernel.Language,
) app.RateSetResult {
	t.Helper()

	result, err := rateSets.RateSet(t.Context(), date, language)
	if err != nil {
		t.Fatalf("RateSet() error = %v", err)
	}

	return result
}

// NBGRATE-SC-019: An outage of the vendor gives the set read before, with its read instant.
func TestRateSetReturnsAStaleSet(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		rateSets := newCache(t, source)
		today := domain.Today(time.Now())
		readAt := time.Now()

		ask(t, rateSets, today, kernel.LanguageEnglish)

		time.Sleep(20 * time.Minute)
		source.failing.Store(true)
		time.Sleep(100 * time.Minute)

		got := ask(t, rateSets, today, kernel.LanguageEnglish)
		if !got.Stale || !got.ReadAt.Equal(readAt) {
			t.Errorf(
				"RateSet() = stale %v read at %v, want stale and read at %v",
				got.Stale,
				got.ReadAt,
				readAt,
			)
		}
	})
}

// NBGRATE-SC-024: The set of today stays 15 minutes.
func TestRateSetKeepsTheSetOfTodayFifteenMinutes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		rateSets := newCache(t, source)
		today := domain.Today(time.Now())

		for _, wait := range []time.Duration{0, 14 * time.Minute, 2 * time.Minute} {
			time.Sleep(wait)
			ask(t, rateSets, today, kernel.LanguageEnglish)
		}

		if got := source.calls.Load(); got != 2 {
			t.Errorf("the source counted %d calls, want 2", got)
		}
	})
}

type callCase struct {
	name string
	run  func(t *testing.T, rateSets *cache.RateSetCache, today domain.Date)
	want int32
}

func concurrentAsks(t *testing.T, rateSets *cache.RateSetCache, today domain.Date) {
	t.Helper()

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { ask(t, rateSets, today, kernel.LanguageEnglish) })
	}

	wg.Wait()
}

func pastDateAfter(wait time.Duration) func(*testing.T, *cache.RateSetCache, domain.Date) {
	return func(t *testing.T, rateSets *cache.RateSetCache, today domain.Date) {
		t.Helper()
		askTwice(t, rateSets, today.AddDays(-5), wait)
	}
}

func twoLanguages(t *testing.T, rateSets *cache.RateSetCache, today domain.Date) {
	t.Helper()
	ask(t, rateSets, today, kernel.LanguageEnglish)
	ask(t, rateSets, today, kernel.LanguageGeorgian)
}

// NBGRATE-SC-025: One call for concurrent asks, one call in 30 days for a past date, and
// one call for each language.
func TestRateSetLimitsTheCalls(t *testing.T) {
	t.Parallel()

	tests := []callCase{
		{name: "concurrent asks", run: concurrentAsks, want: 1},
		{name: "past date at 29 days", run: pastDateAfter(29 * day), want: 1},
		{name: "past date at 31 days", run: pastDateAfter(31 * day), want: 2},
		{name: "two languages", run: twoLanguages, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingSource{}
				rateSets := newCache(t, source)

				tt.run(t, rateSets, domain.Today(time.Now()))

				if got := source.calls.Load(); got != tt.want {
					t.Errorf("the source counted %d calls, want %d", got, tt.want)
				}
			})
		})
	}
}

func askTwice(t *testing.T, rateSets *cache.RateSetCache, date domain.Date, wait time.Duration) {
	t.Helper()

	ask(t, rateSets, date, kernel.LanguageEnglish)
	time.Sleep(wait)
	ask(t, rateSets, date, kernel.LanguageEnglish)
}

// NBGRATE-SC-033: A past set stays stale for 7 days past its 30 days, then it is gone.
func TestRateSetDropsAPastSetAfterItsStaleAge(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingSource{}
		rateSets := newCache(t, source)
		past := domain.Today(time.Now()).AddDays(-5)
		readAt := time.Now()

		ask(t, rateSets, past, kernel.LanguageEnglish)
		source.failing.Store(true)

		time.Sleep(30*day + time.Hour)

		if got := ask(
			t,
			rateSets,
			past,
			kernel.LanguageEnglish,
		); !got.Stale ||
			!got.ReadAt.Equal(readAt) {
			t.Errorf(
				"at 30 days: stale %v read at %v, want stale and read at %v",
				got.Stale,
				got.ReadAt,
				readAt,
			)
		}

		time.Sleep(7 * day)

		if _, err := rateSets.RateSet(
			t.Context(),
			past,
			kernel.LanguageEnglish,
		); !errors.Is(
			err,
			kernel.ErrSourceUnavailable,
		) {
			t.Errorf("at 37 days: error = %v, want ErrSourceUnavailable", err)
		}
	})
}

// NBGRATE-SC-034: Close stops both stores, and a second Close returns no error.
func TestCloseTwice(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		rateSets, err := cache.New(&countingSource{}, 64, time.Now)
		if err != nil {
			t.Fatal(err)
		}

		if err := rateSets.Close(); err != nil {
			t.Errorf("first Close() error = %v", err)
		}

		if err := rateSets.Close(); err != nil {
			t.Errorf("second Close() error = %v", err)
		}
	})
}
