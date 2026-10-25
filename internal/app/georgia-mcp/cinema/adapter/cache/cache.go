// Package cache keeps the lists of Cavea in memory, in front of the ports of the context.
// The seats have no cache: a seat is sold at any moment.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/cache"
)

const (
	// listTTL bounds the age of a list. Cavea changes its program at any time.
	listTTL = 15 * time.Minute
	// maxAge bounds the age of a stale list.
	maxAge = 6 * time.Hour

	oneEntry       = 1
	statusEntries  = 2
	searchEntries  = 1024
	movieEntries   = 256
	everyKey       = "all"
	searchKeyStart = "search|"
)

func newStore[V any](entries int) (*cache.Cache[V], error) {
	store, err := cache.New[V](
		cache.Config{TTL: listTTL, MaxStale: maxAge - listTTL, MaxEntries: entries},
	)
	if err != nil {
		return nil, fmt.Errorf("building a store of Cavea: %w", err)
	}

	return store, nil
}

// CinemaCache implements app.CinemaSource in front of another source.
type CinemaCache struct {
	source  app.CinemaSource
	cinemas *cache.Cache[app.CinemaList]
}

var _ app.CinemaSource = (*CinemaCache)(nil)

// NewCinemaCache returns a cache in front of the source.
func NewCinemaCache(source app.CinemaSource) (*CinemaCache, error) {
	cinemas, err := newStore[app.CinemaList](oneEntry)
	if err != nil {
		return nil, err
	}

	return &CinemaCache{source: source, cinemas: cinemas}, nil
}

// Cinemas returns the cached cinemas, or reads them.
func (c *CinemaCache) Cinemas(ctx context.Context) (app.CinemaList, error) {
	result, err := c.cinemas.Get(ctx, everyKey, c.source.Cinemas)
	if err != nil {
		return app.CinemaList{}, fmt.Errorf("reading the cinemas: %w", err)
	}

	list := result.Value
	list.Stale = result.Stale

	return list, nil
}

// Close stops the goroutines of the store.
func (c *CinemaCache) Close() error {
	c.cinemas.Close()

	return nil
}

// MovieCache implements app.MovieSource in front of another source.
type MovieCache struct {
	source app.MovieSource
	movies *cache.Cache[app.MovieList]
	search *cache.Cache[app.MovieList]
}

var _ app.MovieSource = (*MovieCache)(nil)

// NewMovieCache returns a cache in front of the source.
func NewMovieCache(source app.MovieSource) (*MovieCache, error) {
	movies, err := newStore[app.MovieList](statusEntries)
	if err != nil {
		return nil, err
	}

	search, err := newStore[app.MovieList](searchEntries)
	if err != nil {
		movies.Close()

		return nil, err
	}

	return &MovieCache{source: source, movies: movies, search: search}, nil
}

// Movies returns the cached movies of a status, or reads them.
func (c *MovieCache) Movies(ctx context.Context, status domain.MovieStatus) (app.MovieList, error) {
	result, err := c.movies.Get(
		ctx,
		string(status),
		func(ctx context.Context) (app.MovieList, error) {
			return c.source.Movies(ctx, status)
		},
	)
	if err != nil {
		return app.MovieList{}, fmt.Errorf("reading the movies: %w", err)
	}

	return staleMovies(result), nil
}

// Search returns the cached movies of a text, or searches them. The use case trims the
// text, so the key keeps it as it comes.
func (c *MovieCache) Search(ctx context.Context, text string) (app.MovieList, error) {
	result, err := c.search.Get(
		ctx,
		searchKeyStart+text,
		func(ctx context.Context) (app.MovieList, error) {
			return c.source.Search(ctx, text)
		},
	)
	if err != nil {
		return app.MovieList{}, fmt.Errorf("searching the movies: %w", err)
	}

	return staleMovies(result), nil
}

func staleMovies(result cache.Result[app.MovieList]) app.MovieList {
	list := result.Value
	list.Stale = result.Stale

	return list
}

// Close stops the goroutines of the stores.
func (c *MovieCache) Close() error {
	c.movies.Close()
	c.search.Close()

	return nil
}

// SessionCache implements app.SessionSource in front of another source.
type SessionCache struct {
	source   app.SessionSource
	sessions *cache.Cache[app.SessionList]
	byMovie  *cache.Cache[app.SessionList]
}

var _ app.SessionSource = (*SessionCache)(nil)

// NewSessionCache returns a cache in front of the source.
func NewSessionCache(source app.SessionSource) (*SessionCache, error) {
	sessions, err := newStore[app.SessionList](oneEntry)
	if err != nil {
		return nil, err
	}

	byMovie, err := newStore[app.SessionList](movieEntries)
	if err != nil {
		sessions.Close()

		return nil, err
	}

	return &SessionCache{source: source, sessions: sessions, byMovie: byMovie}, nil
}

// Sessions returns the cached sessions of every movie, or reads them.
func (c *SessionCache) Sessions(ctx context.Context) (app.SessionList, error) {
	result, err := c.sessions.Get(ctx, everyKey, c.source.Sessions)
	if err != nil {
		return app.SessionList{}, fmt.Errorf("reading the sessions: %w", err)
	}

	return staleSessions(result), nil
}

// MovieSessions returns the cached sessions of a movie, or reads them.
func (c *SessionCache) MovieSessions(ctx context.Context, movieID string) (app.SessionList, error) {
	result, err := c.byMovie.Get(
		ctx,
		movieID,
		func(ctx context.Context) (app.SessionList, error) {
			return c.source.MovieSessions(ctx, movieID)
		},
	)
	if err != nil {
		return app.SessionList{}, fmt.Errorf("reading the sessions of a movie: %w", err)
	}

	return staleSessions(result), nil
}

func staleSessions(result cache.Result[app.SessionList]) app.SessionList {
	list := result.Value
	list.Stale = result.Stale

	return list
}

// Close stops the goroutines of the stores.
func (c *SessionCache) Close() error {
	c.sessions.Close()
	c.byMovie.Close()

	return nil
}
