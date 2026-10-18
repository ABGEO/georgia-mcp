package cache_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	usdKey = "usd"
	eurKey = "eur"
)

// manySource answers each key with its name, and records each load.
type manySource struct {
	mu      sync.Mutex
	loads   [][]string
	fail    bool
	release chan struct{}
}

func (s *manySource) load(_ context.Context, keys []string) (map[string]string, error) {
	s.mu.Lock()
	s.loads = append(s.loads, slices.Clone(keys))
	fail, release := s.fail, s.release
	s.mu.Unlock()

	if release != nil {
		<-release
	}

	if fail {
		return nil, fmt.Errorf("calling the vendor: %w", kernel.ErrSourceUnavailable)
	}

	values := make(map[string]string, len(keys))
	for _, key := range keys {
		values[key] = key
	}

	return values, nil
}

func (s *manySource) setFail(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fail = fail
}

func (s *manySource) recorded() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.loads)
}

func manyCache(t *testing.T) *cache.Cache[string] {
	t.Helper()

	return newCache[string](
		t,
		cache.Config{TTL: 5 * time.Minute, MaxStale: 55 * time.Minute, MaxEntries: 10},
	)
}

func getMany(
	t *testing.T,
	values *cache.Cache[string],
	source *manySource,
	keys ...string,
) map[string]cache.Result[string] {
	t.Helper()

	results, err := values.GetMany(t.Context(), keys, source.load)
	if err != nil {
		t.Fatalf("GetMany(%v) error = %v", keys, err)
	}

	return results
}

func TestGetManyLoadsTheKeysThatAreNotFresh(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values, source := manyCache(t), &manySource{}
		first := time.Now()

		getMany(t, values, source, usdKey)
		time.Sleep(time.Minute)

		results := getMany(t, values, source, usdKey, eurKey)

		if diff := cmp.Diff([][]string{{usdKey}, {eurKey}}, source.recorded()); diff != "" {
			t.Errorf("loads mismatch (-want +got):\n%s", diff)
		}

		if !results[usdKey].FetchedAt.Equal(first) || results[eurKey].Value != eurKey {
			t.Errorf("GetMany() = %+v, want usd of the first load and eur", results)
		}
	})
}

func TestGetManySharesOneLoadForTheSameKeys(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values, source := manyCache(t), &manySource{release: make(chan struct{})}

		var wg sync.WaitGroup

		for range 10 {
			wg.Go(func() { getMany(t, values, source, usdKey, eurKey) })
		}

		synctest.Wait()
		close(source.release)
		wg.Wait()

		if got := len(source.recorded()); got != 1 {
			t.Errorf("load ran %d times, want 1", got)
		}
	})
}

func TestGetManyReturnsStaleValuesOfEveryKeyOnly(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		values, source := manyCache(t), &manySource{}
		getMany(t, values, source, usdKey)

		time.Sleep(30 * time.Minute)
		source.setFail(true)

		results := getMany(t, values, source, usdKey)
		if !results[usdKey].Stale {
			t.Errorf("GetMany(usd) = %+v, want a stale value", results[usdKey])
		}

		_, err := values.GetMany(t.Context(), []string{usdKey, eurKey}, source.load)
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("GetMany(usd, eur) error = %v, want ErrSourceUnavailable", err)
		}

		time.Sleep(31 * time.Minute)

		_, err = values.GetMany(t.Context(), []string{usdKey}, source.load)
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("GetMany(usd) past the stale age error = %v, want ErrSourceUnavailable", err)
		}
	})
}

func TestGetManyRefusesALoadThatLacksAKey(t *testing.T) {
	t.Parallel()

	values := manyCache(t)

	_, err := values.GetMany(
		t.Context(),
		[]string{usdKey, eurKey},
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{usdKey: usdRate}, nil
		},
	)
	if err == nil {
		t.Error("GetMany() error = nil, want an error for the key eur")
	}
}

func TestGetManyWithTheCacheOff(t *testing.T) {
	t.Parallel()

	values, source := newCache[string](t, cache.Config{}), &manySource{}

	getMany(t, values, source, usdKey)
	getMany(t, values, source, usdKey)

	if got := len(source.recorded()); got != 2 {
		t.Errorf("load ran %d times, want 2", got)
	}
}
