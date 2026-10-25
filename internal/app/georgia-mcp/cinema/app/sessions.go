package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// ScheduledSession is a session with its movie and its cinema.
type ScheduledSession struct {
	Session domain.Session
	Movie   domain.Movie
	Cinema  domain.Cinema
	// Occupancy is empty when Cavea gives no level.
	Occupancy domain.OccupancyLevel
}

// SessionQuery asks the sessions of one date. A nil filter matches every session.
type SessionQuery struct {
	Movie *domain.MovieCode
	// Date is the zero date for today.
	Date   kernel.Date
	Cinema *domain.CinemaCode
	Audio  *domain.AudioLanguage
	Format *domain.Format
	// Limit is the maximum count of sessions. The tool gives the default.
	Limit int
}

// Sessions is the first sessions of a query, with the count of every match.
type Sessions struct {
	Sessions []ScheduledSession
	Total    int
	// Occupancy is false when Cavea gave no level.
	Occupancy   bool
	ReadAt      time.Time
	OccupancyAt time.Time
	Stale       bool
}

// catalog joins a session with its movie and its open cinema.
type catalog struct {
	cinemas map[string]domain.Cinema
	movies  map[string]domain.Movie
	stale   bool
}

func readCatalog(ctx context.Context, cinemas CinemaSource, movies MovieSource) (catalog, error) {
	cinemaList, err := cinemas.Cinemas(ctx)
	if err != nil {
		return catalog{}, fmt.Errorf("reading the cinemas: %w", err)
	}

	result := catalog{
		cinemas: map[string]domain.Cinema{},
		movies:  map[string]domain.Movie{},
		stale:   cinemaList.Stale,
	}

	for _, cinema := range cinemaList.Cinemas {
		result.cinemas[cinema.ID] = cinema
	}

	for _, status := range domain.MovieStatuses() {
		movieList, err := movies.Movies(ctx, status)
		if err != nil {
			return catalog{}, fmt.Errorf("reading the movies: %w", err)
		}

		result.stale = result.stale || movieList.Stale

		for _, movie := range movieList.Movies {
			result.movies[movie.ID] = movie
		}
	}

	return result, nil
}

func (c catalog) schedule(session domain.Session) (ScheduledSession, bool) {
	cinema, open := c.cinemas[session.CinemaID]
	movie, listed := c.movies[session.MovieID]

	return ScheduledSession{Session: session, Movie: movie, Cinema: cinema}, open && listed
}

func (c catalog) movie(code domain.MovieCode) (domain.Movie, bool) {
	for _, movie := range c.movies {
		if movie.Code == code {
			return movie, true
		}
	}

	return domain.Movie{}, false
}

func (c catalog) hasCinema(code domain.CinemaCode) bool {
	for _, cinema := range c.cinemas {
		if cinema.Code == code {
			return true
		}
	}

	return false
}

// ListSessions lists the sessions of Cavea on one date that have not started.
type ListSessions struct {
	cinemas  CinemaSource
	movies   MovieSource
	sessions SessionSource
	seats    SeatSource
	now      func() time.Time
}

// NewListSessions returns the use case.
func NewListSessions(
	cinemas CinemaSource,
	movies MovieSource,
	sessions SessionSource,
	seats SeatSource,
	now func() time.Time,
) *ListSessions {
	return &ListSessions{
		cinemas:  cinemas,
		movies:   movies,
		sessions: sessions,
		seats:    seats,
		now:      now,
	}
}

// List returns the matching sessions up to the limit, in the order of the start, then the
// cinema code, each with its occupancy level.
func (l *ListSessions) List(ctx context.Context, query SessionQuery) (Sessions, error) {
	now := l.now()

	date, err := dateOf(query.Date, now)
	if err != nil {
		return Sessions{}, err
	}

	joined, err := readCatalog(ctx, l.cinemas, l.movies)
	if err != nil {
		return Sessions{}, fmt.Errorf("listing the sessions: %w", err)
	}

	list, err := l.read(ctx, joined, query)
	if err != nil {
		return Sessions{}, err
	}

	matched := match(list.Sessions, joined, query, date, now)

	result := Sessions{
		Sessions: matched,
		Total:    len(matched),
		ReadAt:   list.ReadAt,
		Stale:    list.Stale || joined.stale,
	}
	if query.Limit > 0 && len(matched) > query.Limit {
		result.Sessions = matched[:query.Limit]
	}

	l.addOccupancy(ctx, &result)

	return result, nil
}

func (l *ListSessions) read(
	ctx context.Context,
	joined catalog,
	query SessionQuery,
) (SessionList, error) {
	if query.Cinema != nil && !joined.hasCinema(*query.Cinema) {
		return SessionList{}, fmt.Errorf("%w: %s", ErrUnknownCinema, query.Cinema)
	}

	if query.Movie == nil {
		list, err := l.sessions.Sessions(ctx)
		if err != nil {
			return SessionList{}, fmt.Errorf("listing the sessions: %w", err)
		}

		return list, nil
	}

	movie, found := joined.movie(*query.Movie)
	if !found {
		return SessionList{}, fmt.Errorf("%w: %s", ErrUnknownMovie, query.Movie)
	}

	list, err := l.sessions.MovieSessions(ctx, movie.ID)
	if err != nil {
		return SessionList{}, fmt.Errorf("listing the sessions of %s: %w", query.Movie, err)
	}

	return list, nil
}

// addOccupancy gives each session its level. A failure leaves every session with no
// level, so the list still answers when the sessions start.
func (l *ListSessions) addOccupancy(ctx context.Context, result *Sessions) {
	if len(result.Sessions) == 0 {
		return
	}

	codes := make([]domain.SessionCode, 0, len(result.Sessions))
	for _, item := range result.Sessions {
		codes = append(codes, item.Session.Code)
	}

	levels, readAt, err := l.seats.Occupancy(ctx, codes)
	if err != nil {
		return
	}

	result.Occupancy = true
	result.OccupancyAt = readAt

	for index := range result.Sessions {
		result.Sessions[index].Occupancy = levels[result.Sessions[index].Session.Code]
	}
}

// dateOf returns the date of a query: today for the zero date. It refuses a date before
// today.
func dateOf(date kernel.Date, now time.Time) (kernel.Date, error) {
	today := kernel.Today(now)

	if date.IsZero() {
		return today, nil
	}

	if date.Before(today) {
		return kernel.Date{}, fmt.Errorf("%w: %s", ErrPastDate, date)
	}

	return date, nil
}

// match keeps the known sessions of the date that start after now and match the filters,
// in the order of the start, then the cinema code.
func match(
	sessions []domain.Session,
	joined catalog,
	query SessionQuery,
	date kernel.Date,
	now time.Time,
) []ScheduledSession {
	matched := []ScheduledSession{}

	for _, session := range sessions {
		item, known := joined.schedule(session)
		if known && session.Code.Date() == date && session.Code.Start().After(now) &&
			matches(item, query) {
			matched = append(matched, item)
		}
	}

	slices.SortStableFunc(matched, func(a, b ScheduledSession) int {
		return cmp.Or(
			a.Session.Code.Start().Compare(b.Session.Code.Start()),
			cmp.Compare(a.Cinema.Code.String(), b.Cinema.Code.String()),
		)
	})

	return matched
}

func matches(item ScheduledSession, query SessionQuery) bool {
	switch {
	case query.Movie != nil && item.Movie.Code != *query.Movie:
		return false
	case query.Cinema != nil && item.Cinema.Code != *query.Cinema:
		return false
	case query.Audio != nil && item.Session.Audio != *query.Audio:
		return false
	case query.Format != nil && !slices.Contains(item.Session.Formats, *query.Format):
		return false
	default:
		return true
	}
}
