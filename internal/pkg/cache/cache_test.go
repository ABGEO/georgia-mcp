package cache_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const usdRate = "2.5982"

var errUnknownCurrency = errors.New("currency: unknown code")

func newCache[V any](t *testing.T, cfg cache.Config) *cache.Cache[V] {
	t.Helper()

	values, err := cache.New[V](cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Cleanup(values.Close)

	return values
}

// PLATFORM-SC-019: A value stays until its time to live ends.
func TestGetKeepsTheValueForTheTimeToLive(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values := newCache[int](
			t,
			cache.Config{TTL: time.Hour, MaxStale: time.Hour, MaxEntries: 10},
		)

		var loads atomic.Int32

		load := func(context.Context) (int, error) { return int(loads.Add(1)), nil }

		for _, at := range []time.Duration{0, 59 * time.Minute, 2 * time.Minute} {
			time.Sleep(at)

			if _, err := values.Get(t.Context(), "usd", load); err != nil {
				t.Fatalf("Get() error = %v", err)
			}
		}

		if got := loads.Load(); got != 2 {
			t.Errorf("load ran %d times, want 2", got)
		}
	})
}

// PLATFORM-SC-020: Ten calls for one missing key run one load.
func TestGetLoadsAKeyOneTime(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values := newCache[string](t, cache.Config{TTL: time.Hour, MaxEntries: 10})

		release := make(chan struct{})

		var loads atomic.Int32

		//nolint:unparam // Get takes a load that can fail. This one never does.
		load := func(context.Context) (string, error) {
			loads.Add(1)
			<-release

			return usdRate, nil
		}

		const callers = 10

		var wg sync.WaitGroup

		results := make([]string, callers)

		for i := range callers {
			wg.Go(func() {
				result, err := values.Get(t.Context(), "usd", load)
				if err != nil {
					t.Errorf("Get() error = %v", err)
				}

				results[i] = result.Value
			})
		}

		synctest.Wait()
		close(release)
		wg.Wait()

		if got := loads.Load(); got != 1 {
			t.Errorf("load ran %d times, want 1", got)
		}

		for i, got := range results {
			if got != usdRate {
				t.Errorf("caller %d got %q, want %q", i, got, usdRate)
			}
		}
	})
}

// PLATFORM-SC-043: A caller that joins the shared load of a key after the last load
// ended gets the stored value.
func TestSharedLoadReadsTheStoreAgain(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values := newCache[string](t, cache.Config{TTL: time.Hour, MaxEntries: 10})

		var loads atomic.Int32

		load := func(context.Context) (string, error) {
			loads.Add(1)

			return usdRate, nil
		}

		if _, err := values.Get(t.Context(), usdKey, load); err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		time.Sleep(time.Minute)

		got, err := cache.SharedLoad(t.Context(), values, usdKey, load)
		if err != nil {
			t.Fatalf("SharedLoad() error = %v", err)
		}

		if got != usdRate || loads.Load() != 1 {
			t.Errorf(
				"SharedLoad() = %q after %d loads, want %q after 1",
				got,
				loads.Load(),
				usdRate,
			)
		}
	})
}

// PLATFORM-SC-043: A caller that joins the shared load of the keys of GetMany after the
// last load ended gets the stored values.
func TestSharedLoadManyReadsTheStoreAgain(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values := newCache[string](t, cache.Config{TTL: time.Hour, MaxEntries: 10})
		source := &manySource{}

		getMany(t, values, source, usdKey, eurKey)
		time.Sleep(time.Minute)

		keys := []string{eurKey, usdKey}

		got, err := cache.SharedLoadMany(t.Context(), values, keys, source.load)
		if err != nil {
			t.Fatalf("SharedLoadMany() error = %v", err)
		}

		want := map[string]string{usdKey: usdKey, eurKey: eurKey}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("SharedLoadMany() mismatch (-want +got):\n%s", diff)
		}

		if loads := len(source.recorded()); loads != 1 {
			t.Errorf("the source recorded %d loads, want 1", loads)
		}
	})
}

func staleCache(t *testing.T) (*cache.Cache[string], time.Time) {
	t.Helper()

	values := newCache[string](
		t,
		cache.Config{TTL: time.Hour, MaxStale: 24 * time.Hour, MaxEntries: 10},
	)
	fetchedAt := time.Now()

	if _, err := values.Get(
		t.Context(),
		"usd",
		func(context.Context) (string, error) { return usdRate, nil },
	); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	return values, fetchedAt
}

// PLATFORM-SC-021: An unavailable source gives the stale value inside the maximum stale age.
func TestGetReturnsAStaleValueWhenTheSourceIsUnavailable(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values, fetchedAt := staleCache(t)

		time.Sleep(2 * time.Hour)

		unavailable := func(context.Context) (string, error) {
			return "", fmt.Errorf("calling the vendor: %w", kernel.ErrSourceUnavailable)
		}

		got, err := values.Get(t.Context(), "usd", unavailable)
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}

		if !got.Stale || got.Value != usdRate || !got.FetchedAt.Equal(fetchedAt) {
			t.Errorf("Get() = %+v, want the stale value fetched at %v", got, fetchedAt)
		}
	})
}

// PLATFORM-SC-022: Another failure of the source passes through, with no stale value.
func TestGetReturnsAnotherErrorAsItIs(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values, _ := staleCache(t)

		time.Sleep(2 * time.Hour)

		_, err := values.Get(
			t.Context(),
			"usd",
			func(context.Context) (string, error) { return "", errUnknownCurrency },
		)
		if !errors.Is(err, errUnknownCurrency) {
			t.Errorf("Get() error = %v, want %v", err, errUnknownCurrency)
		}
	})
}

// PLATFORM-SC-023: A value older than its maximum stale age is gone.
func TestGetReturnsTheErrorPastTheMaximumStaleAge(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values, _ := staleCache(t)

		time.Sleep(26 * time.Hour)

		_, err := values.Get(t.Context(), "usd", func(context.Context) (string, error) {
			return "", kernel.ErrSourceUnavailable
		})
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("Get() error = %v, want ErrSourceUnavailable", err)
		}
	})
}

// PLATFORM-SC-024: The store keeps no more than the maximum count of entries.
func TestGetKeepsTheMaximumCountOfEntries(t *testing.T) {
	t.Parallel()

	const maxEntries = 10

	values := newCache[int](t, cache.Config{TTL: time.Hour, MaxEntries: maxEntries})

	for i := range 100 {
		if _, err := values.Get(
			t.Context(),
			strconv.Itoa(i),
			func(context.Context) (int, error) { return i, nil },
		); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
	}

	if got := cache.EntryCount(values); got > maxEntries {
		t.Errorf("the store holds %d entries, want %d or fewer", got, maxEntries)
	}
}
