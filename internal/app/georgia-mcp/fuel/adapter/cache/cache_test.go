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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingCompany answers an empty list read at the current instant, and counts the calls.
type countingCompany struct {
	calls   atomic.Int32
	failing atomic.Bool
	release chan struct{}
}

func (c *countingCompany) Company() domain.Company {
	return domain.Company{Code: "wissol", Name: "Wissol"}
}

func (c *countingCompany) Prices(context.Context) (app.PriceList, error) {
	c.calls.Add(1)

	if c.release != nil {
		<-c.release
	}

	if c.failing.Load() {
		return app.PriceList{}, fmt.Errorf("calling Wissol: %w", kernel.ErrSourceUnavailable)
	}

	return app.PriceList{ReadAt: time.Now()}, nil
}

func newCache(t *testing.T, source app.PriceSource) *cache.PriceCache {
	t.Helper()

	prices, err := cache.New(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = prices.Close() })

	return prices
}

type keepCase struct {
	name      string
	failsFrom time.Duration
	askedAt   time.Duration
	wantRead  time.Duration
	wantStale bool
	wantCalls int32
	wantErr   error
}

func keepCases() []keepCase {
	return []keepCase{
		{name: "fresh", failsFrom: -1, askedAt: 29 * time.Minute, wantCalls: 1},
		{
			name:      "expired",
			failsFrom: -1,
			askedAt:   31 * time.Minute,
			wantRead:  31 * time.Minute,
			wantCalls: 2,
		},
		{
			name:      "stale",
			failsFrom: time.Minute,
			askedAt:   11 * time.Hour,
			wantStale: true,
			wantCalls: 2,
		},
		{
			name:      "too old",
			failsFrom: time.Minute,
			askedAt:   13 * time.Hour,
			wantCalls: 2,
			wantErr:   kernel.ErrSourceUnavailable,
		},
	}
}

// WISFUEL-SC-014: A list is fresh for 30 minutes. While the company fails, the cache
// returns the stale list for 12 hours after its read.
func TestPriceCacheKeepsAList(t *testing.T) {
	t.Parallel()

	for _, tt := range keepCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				company := &countingCompany{}
				prices := newCache(t, company)

				if _, err := prices.Prices(t.Context()); err != nil {
					t.Fatalf("Prices() at 0 error = %v", err)
				}

				if tt.failsFrom >= 0 {
					time.Sleep(tt.failsFrom)
					company.failing.Store(true)
				}

				time.Sleep(time.Until(start.Add(tt.askedAt)))

				list, err := prices.Prices(t.Context())
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Prices() error = %v, want %v", err, tt.wantErr)
				}

				if calls := company.calls.Load(); calls != tt.wantCalls {
					t.Errorf("calls = %d, want %d", calls, tt.wantCalls)
				}

				if tt.wantErr != nil {
					return
				}

				if !list.ReadAt.Equal(start.Add(tt.wantRead)) || list.Stale != tt.wantStale {
					t.Errorf(
						"ReadAt = %v, Stale = %v, want %v and %v",
						list.ReadAt.Sub(start), list.Stale, tt.wantRead, tt.wantStale,
					)
				}
			})
		})
	}
}

// WISFUEL-SC-015: Asks that arrive at once share one call.
func TestPriceCacheSharesOneCall(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		company := &countingCompany{release: make(chan struct{})}
		prices := newCache(t, company)

		var wg sync.WaitGroup

		for range 10 {
			wg.Go(func() {
				if _, err := prices.Prices(t.Context()); err != nil {
					t.Errorf("Prices() error = %v", err)
				}
			})
		}

		synctest.Wait()
		close(company.release)
		wg.Wait()

		if calls := company.calls.Load(); calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})
}
