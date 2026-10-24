package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	day   = 24 * time.Hour
	query = "რუსთავი"
)

// countingPost answers an empty read at the current instant, counts each call, and
// fails with an outage once failing is set.
type countingPost struct {
	calls   atomic.Int32
	failing atomic.Bool
}

func (c *countingPost) Addresses(
	context.Context,
	string,
	kernel.Language,
) (app.AddressList, error) {
	c.calls.Add(1)

	if c.failing.Load() {
		return app.AddressList{}, fmt.Errorf(
			"calling Georgian Post: %w",
			kernel.ErrSourceUnavailable,
		)
	}

	return app.AddressList{ReadAt: time.Now()}, nil
}

func newCache(t *testing.T, source app.PostalCodeSource) *cache.AddressCache {
	t.Helper()

	addresses, err := cache.NewAddressCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = addresses.Close() })

	return addresses
}

// POSTCODE-SC-016: A list stays in the cache for 7 days. After that, the cache reads
// Georgian Post again.
func TestAddressCacheKeepsAListForSevenDays(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingPost{}
		addresses := newCache(t, source)

		if _, err := addresses.Addresses(t.Context(), query, kernel.LanguageGeorgian); err != nil {
			t.Fatal(err)
		}

		time.Sleep(7*day - time.Hour)

		if _, err := addresses.Addresses(t.Context(), query, kernel.LanguageGeorgian); err != nil ||
			source.calls.Load() != 1 {
			t.Fatalf("Addresses() at 6 days 23 hours = %v with %d calls, want 1 call",
				err, source.calls.Load())
		}

		time.Sleep(time.Hour)

		if _, err := addresses.Addresses(t.Context(), query, kernel.LanguageGeorgian); err != nil ||
			source.calls.Load() != 2 {
			t.Fatalf("Addresses() at 7 days = %v with %d calls, want 2 calls",
				err, source.calls.Load())
		}
	})
}

// POSTCODE-SC-017: While Georgian Post fails, the cache returns the list as stale until
// it is 30 days old. After that, it returns the error.
func TestAddressCacheKeepsAStaleListForThirtyDays(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingPost{}
		addresses := newCache(t, source)

		first, err := addresses.Addresses(t.Context(), query, kernel.LanguageGeorgian)
		if err != nil {
			t.Fatal(err)
		}

		source.failing.Store(true)
		time.Sleep(8 * day)

		stale, err := addresses.Addresses(t.Context(), query, kernel.LanguageGeorgian)
		if err != nil || !stale.Stale || !stale.ReadAt.Equal(first.ReadAt) {
			t.Fatalf(
				"Addresses() at 8 days = stale %t, read %v, error %v, want the stale list of %v",
				stale.Stale,
				stale.ReadAt,
				err,
				first.ReadAt,
			)
		}

		time.Sleep(22*day + time.Second)

		if _, err := addresses.Addresses(t.Context(), query, kernel.LanguageGeorgian); !errors.Is(
			err, kernel.ErrSourceUnavailable,
		) {
			t.Errorf(
				"Addresses() at 30 days and 1 second error = %v, want kernel.ErrSourceUnavailable",
				err,
			)
		}
	})
}
