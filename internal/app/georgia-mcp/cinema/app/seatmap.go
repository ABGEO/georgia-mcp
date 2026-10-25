package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
)

// ClassCount is the seats of one class in a hall, with the price of the class.
type ClassCount struct {
	Class domain.SeatClass
	// Price is nil when the session has none for the class.
	Price *domain.Price
	Seats int
	Free  int
}

// SeatMap is the seats of the hall of one session.
type SeatMap struct {
	Session ScheduledSession
	// Seats are in the order of Cavea.
	Seats []domain.Seat
	// Classes are the classes of the hall, in the order of domain.SeatClasses.
	Classes []ClassCount
	ReadAt  time.Time
}

// GetSeatMap reads the seat map of a session of Cavea that has not started.
type GetSeatMap struct {
	cinemas  CinemaSource
	movies   MovieSource
	sessions SessionSource
	seats    SeatSource
	now      func() time.Time
}

// NewGetSeatMap returns the use case.
func NewGetSeatMap(
	cinemas CinemaSource,
	movies MovieSource,
	sessions SessionSource,
	seats SeatSource,
	now func() time.Time,
) *GetSeatMap {
	return &GetSeatMap{cinemas: cinemas, movies: movies, sessions: sessions, seats: seats, now: now}
}

// Get returns the seats of the session, the free seats of each class, and the price of
// each class.
func (g *GetSeatMap) Get(ctx context.Context, code domain.SessionCode) (SeatMap, error) {
	if !code.Start().After(g.now()) {
		return SeatMap{}, fmt.Errorf("%w: %s", ErrSessionStarted, code)
	}

	item, err := g.session(ctx, code)
	if err != nil {
		return SeatMap{}, err
	}

	seats, readAt, err := g.seats.SeatMap(ctx, code)
	if err != nil {
		return SeatMap{}, fmt.Errorf("reading the seats of %s: %w", code, err)
	}

	return SeatMap{
		Session: item,
		Seats:   seats,
		Classes: countClasses(seats, item.Session.Prices),
		ReadAt:  readAt,
	}, nil
}

// session finds the session in the list of every movie, then takes the price of each
// class from the list of its movie.
func (g *GetSeatMap) session(
	ctx context.Context,
	code domain.SessionCode,
) (ScheduledSession, error) {
	joined, err := readCatalog(ctx, g.cinemas, g.movies)
	if err != nil {
		return ScheduledSession{}, fmt.Errorf("reading the seat map: %w", err)
	}

	all, err := g.sessions.Sessions(ctx)
	if err != nil {
		return ScheduledSession{}, fmt.Errorf("reading the sessions: %w", err)
	}

	index := slices.IndexFunc(
		all.Sessions,
		func(session domain.Session) bool { return session.Code == code },
	)
	if index < 0 {
		return ScheduledSession{}, fmt.Errorf("%w: %s", ErrUnknownSession, code)
	}

	item, known := joined.schedule(all.Sessions[index])
	if !known {
		return ScheduledSession{}, fmt.Errorf("%w: %s", ErrUnknownSession, code)
	}

	priced, err := g.sessions.MovieSessions(ctx, item.Session.MovieID)
	if err != nil {
		return ScheduledSession{}, fmt.Errorf("reading the prices of %s: %w", code, err)
	}

	if index := slices.IndexFunc(priced.Sessions, func(session domain.Session) bool {
		return session.Code == code
	}); index >= 0 {
		item.Session.Prices = priced.Sessions[index].Prices
	}

	return item, nil
}

func countClasses(seats []domain.Seat, prices map[domain.SeatClass]domain.Price) []ClassCount {
	counts := []ClassCount{}

	for _, class := range domain.SeatClasses() {
		count := ClassCount{Class: class}
		if price, ok := prices[class]; ok {
			count.Price = &price
		}

		for _, seat := range seats {
			if seat.Class != class {
				continue
			}

			count.Seats++

			if !seat.Taken {
				count.Free++
			}
		}

		if count.Seats > 0 {
			counts = append(counts, count)
		}
	}

	return counts
}
