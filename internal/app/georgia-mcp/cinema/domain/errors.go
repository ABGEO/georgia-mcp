package domain

import "errors"

var (
	// ErrInvalidCinemaCode tells that a text is not 1 to 40 of a to z, 0 to 9, and _.
	ErrInvalidCinemaCode = errors.New("cinema code: invalid")
	// ErrInvalidMovieCode tells that a text is not 1 to 100 of a to z, 0 to 9, and -.
	ErrInvalidMovieCode = errors.New("movie code: invalid")
	// ErrInvalidSessionCode tells that a text is not <hall key>/DD-MM-YYYY/HH:MM.
	ErrInvalidSessionCode = errors.New("session code: invalid")
	// ErrInvalidPrice tells that a text is not a decimal amount greater than zero.
	ErrInvalidPrice = errors.New("price: invalid")
	// ErrUnknownMovieStatus tells that a text is not a movie status.
	ErrUnknownMovieStatus = errors.New("movie status: unknown")
	// ErrUnknownAudioLanguage tells that a text is not an audio language.
	ErrUnknownAudioLanguage = errors.New("audio language: unknown")
	// ErrUnknownFormat tells that a text is not a format.
	ErrUnknownFormat = errors.New("format: unknown")
	// ErrUnknownSeatClass tells that a text is not a seat class.
	ErrUnknownSeatClass = errors.New("seat class: unknown")
)
