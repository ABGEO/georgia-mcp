package cache_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingGNERC answers an empty book, counts each call of each method, and fails with
// err once it is set.
type countingGNERC struct {
	balance, prices atomic.Int32
	err             atomic.Pointer[error]
}

func (c *countingGNERC) Balance(context.Context) (app.BalanceBook, error) {
	c.balance.Add(1)

	if err := c.err.Load(); err != nil {
		return app.BalanceBook{}, *err
	}

	return app.BalanceBook{ReadAt: time.Now()}, nil
}

func (c *countingGNERC) Prices(context.Context) (app.PriceBook, error) {
	c.prices.Add(1)

	if err := c.err.Load(); err != nil {
		return app.PriceBook{}, *err
	}

	return app.PriceBook{ReadAt: time.Now()}, nil
}

func (c *countingGNERC) fail(err error) {
	c.err.Store(&err)
}

type bookRead struct {
	read  func() (bool, error)
	calls func() int32
}

func bookReads(t *testing.T, source *countingGNERC) map[string]bookRead {
	t.Helper()

	books, err := cache.NewBookCache(source, source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = books.Close() })

	return map[string]bookRead{
		"Balance": {
			read: func() (bool, error) {
				book, err := books.Balance(t.Context())

				return book.Stale, err
			},
			calls: func() int32 { return source.balance.Load() },
		},
		"Prices": {
			read: func() (bool, error) {
				book, err := books.Prices(t.Context())

				return book.Stale, err
			},
			calls: func() int32 { return source.prices.Load() },
		},
	}
}

// GNERCMON-SC-018: A book stays in the cache for 6 hours.
func TestBookCacheKeepsABookFor6Hours(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		reads := bookReads(t, &countingGNERC{})

		for _, wait := range []time.Duration{0, 5*time.Hour + 59*time.Minute, time.Minute} {
			time.Sleep(wait)

			for name, book := range reads {
				if _, err := book.read(); err != nil {
					t.Fatalf("%s() error = %v", name, err)
				}
			}

			want := int32(1)
			if wait == time.Minute {
				want = 2
			}

			for name, book := range reads {
				if got := book.calls(); got != want {
					t.Errorf("%s calls after %v more = %d, want %d", name, wait, got, want)
				}
			}
		}
	})
}

// GNERCMON-SC-019: While GNERC fails, the cache returns a book as stale until it is 7 days
// old.
func TestBookCacheKeepsAStaleBookFor7Days(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingGNERC{}
		reads := bookReads(t, source)

		for name, book := range reads {
			if _, err := book.read(); err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}
		}

		source.fail(kernel.ErrSourceUnavailable)
		time.Sleep(7*24*time.Hour - time.Minute)

		for name, book := range reads {
			if stale, err := book.read(); err != nil || !stale {
				t.Errorf("%s() at 6 days 23:59 = stale %t, %v, want stale", name, stale, err)
			}
		}

		time.Sleep(2 * time.Minute)

		for name, book := range reads {
			if _, err := book.read(); !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf(
					"%s() at 7 days 00:01 error = %v, want kernel.ErrSourceUnavailable",
					name,
					err,
				)
			}
		}
	})
}
