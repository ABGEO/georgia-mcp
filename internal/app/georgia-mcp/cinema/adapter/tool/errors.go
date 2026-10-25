package tool

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func invalid(err error, format string, args ...any) error {
	return &mcpserver.ToolError{Message: fmt.Sprintf(format, args...), Cause: err}
}

func parseLanguage(text string) (kernel.Language, error) {
	language, err := kernel.ParseLanguage(text)
	if err != nil {
		return "", invalid(err, "language: %q is not a language of this tool. Use ka or en.", text)
	}

	return language, nil
}

// withinRange returns the default for 0, the value inside 1 to the maximum, or an error.
func withinRange(field string, value, fallback, maximum int) (int, error) {
	if value == 0 {
		return fallback, nil
	}

	if value < 1 || value > maximum {
		return 0, invalid(nil, "%s: %d is out of range. Use 1 to %d.", field, value, maximum)
	}

	return value, nil
}

func toSessionQuery(in listSessionsInput) (app.SessionQuery, error) {
	limit, err := withinRange("limit", in.Limit, defaultSessions, maxItems)
	if err != nil {
		return app.SessionQuery{}, err
	}

	query := app.SessionQuery{Limit: limit}

	if in.Date != "" {
		if query.Date, err = kernel.ParseDate(in.Date); err != nil {
			return app.SessionQuery{}, invalid(
				err,
				"date: %q is not a date. Use YYYY-MM-DD.",
				in.Date,
			)
		}
	}

	if query.Movie, err = optional(in.Movie, domain.ParseMovieCode); err != nil {
		return app.SessionQuery{}, unknownMovie(err, in.Movie)
	}

	if query.Cinema, err = optional(in.Cinema, domain.ParseCinemaCode); err != nil {
		return app.SessionQuery{}, unknownCinema(err, in.Cinema)
	}

	if query.Audio, err = optional(in.AudioLanguage, domain.ParseAudioLanguage); err != nil {
		return app.SessionQuery{}, invalid(err,
			"audio_language: %q is not a language of Cavea. Use ka or en.", in.AudioLanguage)
	}

	if query.Format, err = optional(in.Format, domain.ParseFormat); err != nil {
		return app.SessionQuery{}, invalid(err, "format: %q is not a format. Use 3d, imax, "+
			"dolby_atmos, screenx, or infinity_vision.", in.Format)
	}

	return query, nil
}

// optional parses a text that the request can leave out. An empty text gives no value.
func optional[T any](text string, parse func(string) (T, error)) (*T, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No value is the meaning of an empty text.
	}

	value, err := parse(text)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func unknownMovie(err error, movie string) error {
	return invalid(err, "movie: %q is not a movie of Cavea. Use cinema_list_movies or "+
		"cinema_search_movies.", movie)
}

func unknownCinema(err error, cinema string) error {
	return invalid(err, "cinema: %q is not a cinema of Cavea. Use cinema_list_cinemas.", cinema)
}

func sessionsError(err error, in listSessionsInput) error {
	switch {
	case errors.Is(err, app.ErrPastDate):
		return invalid(err, "date: %s is before today. Use today or a later date.", in.Date)
	case errors.Is(err, app.ErrUnknownMovie):
		return unknownMovie(err, in.Movie)
	case errors.Is(err, app.ErrUnknownCinema):
		return unknownCinema(err, in.Cinema)
	default:
		return toolError(err)
	}
}

func seatMapError(err error, session string) error {
	switch {
	case errors.Is(err, app.ErrUnknownSession):
		return invalid(
			err,
			"session: %q is not a session of Cavea. Use cinema_list_sessions.",
			session,
		)
	case errors.Is(err, app.ErrSessionStarted):
		return invalid(
			err,
			"session: %q started. Use cinema_list_sessions for a later session.",
			session,
		)
	default:
		return toolError(err)
	}
}

// toolError gives the model a message that names the cause and the next action. It never
// repeats the text of a Go error or of Cavea.
func toolError(err error) error {
	switch {
	case errors.Is(err, app.ErrInvalidQuery):
		return invalid(err, "query: give 2 to 100 characters, in English.")
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return invalid(err, "Cavea did not answer. Try again in a few minutes.")
	default:
		return invalid(err, "The server failed to read the data of Cavea.")
	}
}
