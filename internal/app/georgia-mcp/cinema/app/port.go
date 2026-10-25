package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
)

// CinemaList is the open cinemas, with the facts of their read.
type CinemaList struct {
	// Cinemas are in the order of Cavea.
	Cinemas []domain.Cinema
	ReadAt  time.Time
	Stale   bool
}

// MovieList is a list of movies, with the facts of its read.
type MovieList struct {
	// Movies are in the order of Cavea.
	Movies []domain.Movie
	ReadAt time.Time
	Stale  bool
}

// SessionList is a list of sessions, with the facts of its read.
type SessionList struct {
	Sessions []domain.Session
	ReadAt   time.Time
	Stale    bool
}

// CinemaSource reads the open cinemas.
type CinemaSource interface {
	Cinemas(ctx context.Context) (CinemaList, error)
}

// MovieSource reads the movies of a status, and the movies that match a text.
type MovieSource interface {
	Movies(ctx context.Context, status domain.MovieStatus) (MovieList, error)
	Search(ctx context.Context, text string) (MovieList, error)
}

// SessionSource reads the sessions of every movie with the price of the class Standard,
// and the sessions of one movie with the price of each class.
type SessionSource interface {
	Sessions(ctx context.Context) (SessionList, error)
	MovieSessions(ctx context.Context, movieID string) (SessionList, error)
}

// SeatSource reads the live state of the seats: the occupancy level of each session,
// and the seat map of one session. It has no cache.
type SeatSource interface {
	Occupancy(
		ctx context.Context,
		codes []domain.SessionCode,
	) (map[domain.SessionCode]domain.OccupancyLevel, time.Time, error)
	SeatMap(ctx context.Context, code domain.SessionCode) ([]domain.Seat, time.Time, error)
}
