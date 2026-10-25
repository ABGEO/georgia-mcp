package app

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
)

const (
	minQueryRunes = 2
	maxQueryRunes = 100
)

// ListCinemas lists the open cinemas of Cavea.
type ListCinemas struct {
	source CinemaSource
}

// NewListCinemas returns the use case.
func NewListCinemas(source CinemaSource) *ListCinemas {
	return &ListCinemas{source: source}
}

// List returns the open cinemas.
func (l *ListCinemas) List(ctx context.Context) (CinemaList, error) {
	list, err := l.source.Cinemas(ctx)
	if err != nil {
		return CinemaList{}, fmt.Errorf("listing the cinemas: %w", err)
	}

	return list, nil
}

// Movies is the first movies of a list, with the count of the full list.
type Movies struct {
	Movies []domain.Movie
	Total  int
	ReadAt time.Time
	Stale  bool
}

func firstMovies(list MovieList, limit int) Movies {
	movies := list.Movies
	if limit > 0 && len(movies) > limit {
		movies = movies[:limit]
	}

	return Movies{Movies: movies, Total: len(list.Movies), ReadAt: list.ReadAt, Stale: list.Stale}
}

// MovieQuery asks the movies of a status.
type MovieQuery struct {
	Status domain.MovieStatus
	// Limit is the maximum count of movies. The tool gives the default.
	Limit int
}

// ListMovies lists the movies of Cavea now showing or coming soon.
type ListMovies struct {
	source MovieSource
}

// NewListMovies returns the use case.
func NewListMovies(source MovieSource) *ListMovies {
	return &ListMovies{source: source}
}

// List returns the movies of the status up to the limit.
func (l *ListMovies) List(ctx context.Context, query MovieQuery) (Movies, error) {
	list, err := l.source.Movies(ctx, query.Status)
	if err != nil {
		return Movies{}, fmt.Errorf("listing the movies: %w", err)
	}

	return firstMovies(list, query.Limit), nil
}

// SearchQuery asks the movies that match a text.
type SearchQuery struct {
	Text string
	// Limit is the maximum count of movies. The tool gives the default.
	Limit int
}

// SearchMovies finds the movies of Cavea by a part of the English name.
type SearchMovies struct {
	source MovieSource
}

// NewSearchMovies returns the use case.
func NewSearchMovies(source MovieSource) *SearchMovies {
	return &SearchMovies{source: source}
}

// Search returns the matching movies up to the limit.
func (s *SearchMovies) Search(ctx context.Context, query SearchQuery) (Movies, error) {
	text := strings.TrimSpace(query.Text)
	if count := utf8.RuneCountInString(text); count < minQueryRunes || count > maxQueryRunes {
		return Movies{}, fmt.Errorf("%w: %d characters", ErrInvalidQuery, count)
	}

	list, err := s.source.Search(ctx, text)
	if err != nil {
		return Movies{}, fmt.Errorf("searching the movies: %w", err)
	}

	return firstMovies(list, query.Limit), nil
}
