package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
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
	now    func() time.Time
}

// NewListMovies returns the use case.
func NewListMovies(source MovieSource, now func() time.Time) *ListMovies {
	return &ListMovies{source: source, now: now}
}

// List returns the movies of the status up to the limit. A movie now showing before its
// premiere is coming soon, after the movies of Cavea coming soon.
func (l *ListMovies) List(ctx context.Context, query MovieQuery) (Movies, error) {
	showing, err := l.source.Movies(ctx, domain.NowShowing)
	if err != nil {
		return Movies{}, fmt.Errorf("listing the movies: %w", err)
	}

	today := kernel.Today(l.now())
	result := MovieList{ReadAt: showing.ReadAt, Stale: showing.Stale}
	moved := []domain.Movie{}

	for _, movie := range showing.Movies {
		if beforePremiere(movie, today) {
			movie.Status = domain.ComingSoon
			moved = append(moved, movie)
		} else if query.Status == domain.NowShowing {
			result.Movies = append(result.Movies, movie)
		}
	}

	if query.Status == domain.ComingSoon {
		soon, err := l.source.Movies(ctx, domain.ComingSoon)
		if err != nil {
			return Movies{}, fmt.Errorf("listing the movies: %w", err)
		}

		result.Movies = slices.Concat(soon.Movies, moved)
		result.ReadAt = older(result.ReadAt, soon.ReadAt)
		result.Stale = result.Stale || soon.Stale
	}

	return firstMovies(result, query.Limit), nil
}

// beforePremiere reports a movie whose premiere is a full date after today.
func beforePremiere(movie domain.Movie, today kernel.Date) bool {
	premiere, err := kernel.ParseDate(movie.Premiere)

	return err == nil && premiere.After(today)
}

func older(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}

	return a
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
	now    func() time.Time
}

// NewSearchMovies returns the use case.
func NewSearchMovies(source MovieSource, now func() time.Time) *SearchMovies {
	return &SearchMovies{source: source, now: now}
}

// Search returns the matching movies up to the limit. The search of Cavea gives no
// premiere, so a found movie takes it from the list now showing.
func (s *SearchMovies) Search(ctx context.Context, query SearchQuery) (Movies, error) {
	text := strings.TrimSpace(query.Text)
	if count := utf8.RuneCountInString(text); count < minQueryRunes || count > maxQueryRunes {
		return Movies{}, fmt.Errorf("%w: %d characters", ErrInvalidQuery, count)
	}

	list, err := s.source.Search(ctx, text)
	if err != nil {
		return Movies{}, fmt.Errorf("searching the movies: %w", err)
	}

	showing, err := s.source.Movies(ctx, domain.NowShowing)
	if err != nil {
		return Movies{}, fmt.Errorf("reading the premieres of the movies: %w", err)
	}

	today := kernel.Today(s.now())
	list.Movies = slices.Clone(list.Movies)

	for index, found := range list.Movies {
		at := slices.IndexFunc(
			showing.Movies,
			func(movie domain.Movie) bool { return movie.ID == found.ID },
		)
		if at >= 0 && beforePremiere(showing.Movies[at], today) {
			list.Movies[index].Status = domain.ComingSoon
		}
	}

	list.ReadAt = older(list.ReadAt, showing.ReadAt)
	list.Stale = list.Stale || showing.Stale

	return firstMovies(list, query.Limit), nil
}
