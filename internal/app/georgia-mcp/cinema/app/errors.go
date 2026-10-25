package app

import "errors"

var (
	// ErrInvalidQuery tells that a search text has fewer than 2 or more than 100 characters.
	ErrInvalidQuery = errors.New("query: invalid")
	// ErrPastDate tells that a date is before today in Tbilisi.
	ErrPastDate = errors.New("date: before today")
	// ErrUnknownMovie tells that Cavea lists no movie with a code.
	ErrUnknownMovie = errors.New("movie: unknown")
	// ErrUnknownCinema tells that Cavea lists no open cinema with a code.
	ErrUnknownCinema = errors.New("cinema: unknown")
	// ErrUnknownSession tells that Cavea lists no session with a code.
	ErrUnknownSession = errors.New("session: unknown")
	// ErrSessionStarted tells that a session started before now.
	ErrSessionStarted = errors.New("session: started")
)
