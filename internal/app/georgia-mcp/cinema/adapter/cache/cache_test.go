package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingCavea answers empty lists read at the current instant, counts each call, and
// fails with an outage once failing is set.
type countingCavea struct {
	calls   atomic.Int32
	failing atomic.Bool
}

func (c *countingCavea) Cinemas(context.Context) (app.CinemaList, error) {
	readAt, err := c.read()

	return app.CinemaList{ReadAt: readAt}, err
}

func (c *countingCavea) Movies(context.Context, domain.MovieStatus) (app.MovieList, error) {
	readAt, err := c.read()

	return app.MovieList{ReadAt: readAt}, err
}

func (c *countingCavea) Search(context.Context, string) (app.MovieList, error) {
	readAt, err := c.read()

	return app.MovieList{ReadAt: readAt}, err
}

func (c *countingCavea) Sessions(context.Context) (app.SessionList, error) {
	readAt, err := c.read()

	return app.SessionList{ReadAt: readAt}, err
}

func (c *countingCavea) MovieSessions(context.Context, string) (app.SessionList, error) {
	readAt, err := c.read()

	return app.SessionList{ReadAt: readAt}, err
}

func (c *countingCavea) read() (time.Time, error) {
	c.calls.Add(1)

	if c.failing.Load() {
		return time.Time{}, fmt.Errorf("calling Cavea: %w", kernel.ErrSourceUnavailable)
	}

	return time.Now(), nil
}

// read is one cached method: it returns the stale mark of its answer.
type read func(ctx context.Context) (bool, error)

// methods builds the three caches in front of one source, and returns one read of each
// cached method.
func methods(t *testing.T, source *countingCavea) map[string]read {
	t.Helper()

	cinemas, err := cache.NewCinemaCache(source)
	if err != nil {
		t.Fatal(err)
	}

	movies, err := cache.NewMovieCache(source)
	if err != nil {
		t.Fatal(err)
	}

	sessions, err := cache.NewSessionCache(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = cinemas.Close()
		_ = movies.Close()
		_ = sessions.Close()
	})

	return map[string]read{
		"Cinemas": func(ctx context.Context) (bool, error) {
			list, err := cinemas.Cinemas(ctx)

			return list.Stale, err
		},
		"Movies": func(ctx context.Context) (bool, error) {
			list, err := movies.Movies(ctx, domain.NowShowing)

			return list.Stale, err
		},
		"Search": func(ctx context.Context) (bool, error) {
			list, err := movies.Search(ctx, "street")

			return list.Stale, err
		},
		"Sessions": func(ctx context.Context) (bool, error) {
			list, err := sessions.Sessions(ctx)

			return list.Stale, err
		},
		"MovieSessions": func(ctx context.Context) (bool, error) {
			list, err := sessions.MovieSessions(ctx, "6ab27d3fde5280679e341878")

			return list.Stale, err
		},
	}
}

// methodNames returns the name of each cached method.
func methodNames() []string {
	return []string{"Cinemas", "Movies", "Search", "Sessions", "MovieSessions"}
}

// CAVCINE-SC-020: Each list stays in the cache for 15 minutes. After that, the cache
// reads Cavea again.
func TestEachListStaysForFifteenMinutes(t *testing.T) {
	t.Parallel()

	for _, name := range methodNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingCavea{}
				call := methods(t, source)[name]

				if _, err := call(t.Context()); err != nil {
					t.Fatal(err)
				}

				time.Sleep(15*time.Minute - time.Second)

				if _, err := call(t.Context()); err != nil || source.calls.Load() != 1 {
					t.Fatalf("at 14:59 = %v with %d calls, want 1 call", err, source.calls.Load())
				}

				time.Sleep(time.Second)

				if _, err := call(t.Context()); err != nil || source.calls.Load() != 2 {
					t.Fatalf("at 15:00 = %v with %d calls, want 2 calls", err, source.calls.Load())
				}
			})
		})
	}
}

// CAVCINE-SC-021: While Cavea fails, each list stays as stale until it is 6 hours old.
// After that, the cache returns the error.
func TestEachListStaysStaleForSixHours(t *testing.T) {
	t.Parallel()

	for _, name := range methodNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				source := &countingCavea{}
				call := methods(t, source)[name]

				if _, err := call(t.Context()); err != nil {
					t.Fatal(err)
				}

				source.failing.Store(true)
				time.Sleep(time.Hour)

				if stale, err := call(t.Context()); err != nil || !stale {
					t.Fatalf("at 1 hour = stale %t, error %v, want the stale list", stale, err)
				}

				time.Sleep(5*time.Hour + time.Second)

				if _, err := call(t.Context()); !errors.Is(err, kernel.ErrSourceUnavailable) {
					t.Errorf(
						"at 6 hours and 1 second error = %v, want kernel.ErrSourceUnavailable",
						err,
					)
				}
			})
		})
	}
}
