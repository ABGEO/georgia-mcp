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

// countingGSE answers an empty read at the current instant, counts each call, and fails
// with err once it is set.
type countingGSE struct {
	calls atomic.Int32
	err   atomic.Pointer[error]
}

func (c *countingGSE) Newest(context.Context) (app.GridRead, error) {
	return c.answer()
}

func (c *countingGSE) At(context.Context, time.Time) (app.GridRead, error) {
	return c.answer()
}

func (c *countingGSE) answer() (app.GridRead, error) {
	c.calls.Add(1)

	if err := c.err.Load(); err != nil {
		return app.GridRead{}, *err
	}

	return app.GridRead{ReadAt: time.Now()}, nil
}

func (c *countingGSE) fail(err error) {
	c.err.Store(&err)
}

func newGridCache(t *testing.T, source app.GridSource) *cache.GridCache {
	t.Helper()

	grid, err := cache.NewGridCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = grid.Close() })

	return grid
}

func methods(t *testing.T, grid *cache.GridCache) map[string]func() (app.GridRead, error) {
	t.Helper()

	slot := time.Date(2026, time.November, 16, 12, 0, 0, 0, kernel.TbilisiZone())

	return map[string]func() (app.GridRead, error){
		"Newest": func() (app.GridRead, error) { return grid.Newest(t.Context()) },
		"At":     func() (app.GridRead, error) { return grid.At(t.Context(), slot) },
	}
}

// GSEGRID-SC-015: A grid read stays in the cache for 1 minute.
func TestGridCacheKeepsAReadFor1Minute(t *testing.T) {
	t.Parallel()

	for name := range methods(t, nil) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingGSE{}
				read := methods(t, newGridCache(t, source))[name]

				for _, wait := range []time.Duration{0, 59 * time.Second} {
					time.Sleep(wait)

					if _, err := read(); err != nil {
						t.Fatal(err)
					}
				}

				if got := source.calls.Load(); got != 1 {
					t.Errorf("calls after 59 seconds = %d, want 1", got)
				}

				time.Sleep(time.Second)

				if _, err := read(); err != nil {
					t.Fatal(err)
				}

				if got := source.calls.Load(); got != 2 {
					t.Errorf("calls after 1 minute = %d, want 2", got)
				}
			})
		})
	}
}

// GSEGRID-SC-016: While GSE fails, the cache returns a read as stale until it is 30
// minutes old.
func TestGridCacheKeepsAStaleReadFor30Minutes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &countingGSE{}
		reads := methods(t, newGridCache(t, source))

		for name, read := range reads {
			if _, err := read(); err != nil {
				t.Fatalf("%s() error = %v", name, err)
			}
		}

		source.fail(kernel.ErrSourceUnavailable)
		time.Sleep(29*time.Minute + 59*time.Second)

		for name, read := range reads {
			if grid, err := read(); err != nil || !grid.Stale {
				t.Errorf("%s() at 29:59 = stale %t, %v, want stale", name, grid.Stale, err)
			}
		}

		time.Sleep(2 * time.Second)

		for name, read := range reads {
			if _, err := read(); !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("%s() at 30:01 error = %v, want kernel.ErrSourceUnavailable", name, err)
			}
		}
	})
}

// GSEGRID-SC-016: ErrNoGrid passes through, and the cache keeps nothing for it.
func TestGridCachePassesNoGridThrough(t *testing.T) {
	t.Parallel()

	source := &countingGSE{}
	source.fail(app.ErrNoGrid)

	for name, read := range methods(t, newGridCache(t, source)) {
		for range 2 {
			if _, err := read(); !errors.Is(err, app.ErrNoGrid) {
				t.Errorf("%s() error = %v, want app.ErrNoGrid", name, err)
			}
		}
	}

	if got := source.calls.Load(); got != 4 {
		t.Errorf("calls = %d, want 4", got)
	}
}
