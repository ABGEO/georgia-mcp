package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const day = 24 * time.Hour

// countingSource counts each call, and fails with failure once it is set.
type countingSource struct {
	calls   atomic.Int32
	failure atomic.Pointer[error]
}

func (c *countingSource) Search(context.Context, domain.SearchQuery) (app.SearchPage, error) {
	return app.SearchPage{ReadAt: time.Now()}, c.answer()
}

func (c *countingSource) Titles(context.Context, string) (app.TitleList, error) {
	return app.TitleList{ReadAt: time.Now()}, c.answer()
}

func (c *countingSource) Act(
	context.Context,
	domain.ActID,
	*int,
	kernel.Language,
) (app.ActText, error) {
	return app.ActText{ReadAt: time.Now()}, c.answer()
}

func (c *countingSource) Status(context.Context, string) (app.StatusResult, error) {
	return app.StatusResult{ReadAt: time.Now()}, c.answer()
}

func (c *countingSource) answer() error {
	c.calls.Add(1)

	if err := c.failure.Load(); err != nil {
		return *err
	}

	return nil
}

func (c *countingSource) fail(err error) {
	c.failure.Store(&err)
}

func newCache(t *testing.T, source app.ActSource) *cache.ActCache {
	t.Helper()

	acts, err := cache.NewActCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = acts.Close() })

	return acts
}

// method calls one method of the cache and returns whether its answer is stale.
type method struct {
	name string
	ttl  time.Duration
	call func(ctx context.Context, acts *cache.ActCache) (bool, error)
}

func methods(t *testing.T) []method {
	t.Helper()

	id, err := domain.ParseActID("1043717")
	if err != nil {
		t.Fatal(err)
	}

	version := 200

	return []method{
		{
			name: "Search",
			ttl:  time.Hour,
			call: func(ctx context.Context, acts *cache.ActCache) (bool, error) {
				page, err := acts.Search(ctx, domain.SearchQuery{Text: "კოდექსი"})

				return page.Stale, err
			},
		},
		{
			name: "Titles",
			ttl:  time.Hour,
			call: func(ctx context.Context, acts *cache.ActCache) (bool, error) {
				list, err := acts.Titles(ctx, "კოდექსი")

				return list.Stale, err
			},
		},
		{
			name: "Status",
			ttl:  time.Hour,
			call: func(ctx context.Context, acts *cache.ActCache) (bool, error) {
				result, err := acts.Status(ctx, "200000000.05.001.016012")

				return result.Stale, err
			},
		},
		{
			name: "Act newest",
			ttl:  time.Hour,
			call: func(ctx context.Context, acts *cache.ActCache) (bool, error) {
				text, err := acts.Act(ctx, id, nil, kernel.LanguageGeorgian)

				return text.Stale, err
			},
		},
		{
			name: "Act version",
			ttl:  day,
			call: func(ctx context.Context, acts *cache.ActCache) (bool, error) {
				text, err := acts.Act(ctx, id, &version, kernel.LanguageGeorgian)

				return text.Stale, err
			},
		},
	}
}

// MATSNE-SC-023: Each method reads again when its answer reaches its time to live.
func TestTheCacheKeepsEachAnswerForItsTime(t *testing.T) {
	t.Parallel()

	for _, item := range methods(t) {
		t.Run(item.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingSource{}
				acts := newCache(t, source)

				if _, err := item.call(t.Context(), acts); err != nil {
					t.Fatal(err)
				}

				time.Sleep(item.ttl - time.Minute)

				if _, err := item.call(t.Context(), acts); err != nil || source.calls.Load() != 1 {
					t.Fatalf(
						"before the time to live: %v with %d calls, want 1",
						err,
						source.calls.Load(),
					)
				}

				time.Sleep(time.Minute)

				if _, err := item.call(t.Context(), acts); err != nil || source.calls.Load() != 2 {
					t.Fatalf(
						"at the time to live: %v with %d calls, want 2",
						err,
						source.calls.Load(),
					)
				}
			})
		})
	}
}

// MATSNE-SC-023: While the source fails, each method returns a stale answer for 7 days.
func TestTheCacheServesAStaleAnswer(t *testing.T) {
	t.Parallel()

	for _, item := range methods(t) {
		t.Run(item.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingSource{}
				acts := newCache(t, source)

				if _, err := item.call(t.Context(), acts); err != nil {
					t.Fatal(err)
				}

				source.fail(fmt.Errorf("down: %w", kernel.ErrSourceUnavailable))
				time.Sleep(7*day - time.Minute)

				if stale, err := item.call(t.Context(), acts); err != nil || !stale {
					t.Fatalf("at 6 days 23:59 = stale %v, %v, want a stale answer", stale, err)
				}

				time.Sleep(time.Minute + time.Second)

				if _, err := item.call(
					t.Context(),
					acts,
				); !errors.Is(
					err,
					kernel.ErrSourceUnavailable,
				) {
					t.Fatalf("after 7 days = %v, want ErrSourceUnavailable", err)
				}
			})
		})
	}
}

// MATSNE-SC-024: An unknown act and a missing translation pass through and are not kept.
func TestTheCacheKeepsNoError(t *testing.T) {
	t.Parallel()

	id, err := domain.ParseActID("999999999")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []error{app.ErrUnknownAct, app.ErrNoTranslation} {
		source := &countingSource{}
		source.fail(want)
		acts := newCache(t, source)

		for range 2 {
			if _, err := acts.Act(
				t.Context(),
				id,
				nil,
				kernel.LanguageEnglish,
			); !errors.Is(
				err,
				want,
			) {
				t.Errorf("Act() error = %v, want %v", err, want)
			}
		}

		if calls := source.calls.Load(); calls != 2 {
			t.Errorf("calls for %v = %d, want 2", want, calls)
		}
	}
}
