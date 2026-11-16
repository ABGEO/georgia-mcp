package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	maxCinemaCode = 40
	maxMovieCode  = 100
	// sessionLayout is the date and the time of a session code, in Tbilisi.
	sessionLayout = "02-01-2006/15:04"
)

// CinemaCode is the short lowercase name of a cinema in a request: "east_point".
type CinemaCode struct {
	code string
}

// ParseCinemaCode returns the cinema code of a text of 1 to 40 of a to z, 0 to 9, and _.
func ParseCinemaCode(text string) (CinemaCode, error) {
	if !isCode(text, maxCinemaCode, isCinemaRune) {
		return CinemaCode{}, fmt.Errorf("%w: %q", ErrInvalidCinemaCode, text)
	}

	return CinemaCode{code: text}, nil
}

// String returns the code: "east_point".
func (c CinemaCode) String() string {
	return c.code
}

// MovieCode is the name of a movie in the address of its page on Cavea: "street-fighter".
type MovieCode struct {
	code string
}

// ParseMovieCode returns the movie code of a text of 1 to 100 of a to z, A to Z, 0 to 9,
// and -. It keeps the case: the page of a movie on Cavea answers only to the code in its case.
func ParseMovieCode(text string) (MovieCode, error) {
	if !isCode(text, maxMovieCode, isMovieRune) {
		return MovieCode{}, fmt.Errorf("%w: %q", ErrInvalidMovieCode, text)
	}

	return MovieCode{code: text}, nil
}

// String returns the code: "street-fighter".
func (c MovieCode) String() string {
	return c.code
}

// Equal reports whether two codes name one movie, with no regard to the case.
func (c MovieCode) Equal(other MovieCode) bool {
	return strings.EqualFold(c.code, other.code)
}

func isCode(text string, maximum int, allowed func(rune) bool) bool {
	if text == "" || len(text) > maximum {
		return false
	}

	for _, r := range text {
		if !allowed(r) {
			return false
		}
	}

	return true
}

func isCinemaRune(r rune) bool {
	return isLowerOrDigit(r) || r == '_'
}

func isMovieRune(r rune) bool {
	return isLowerOrDigit(r) || (r >= 'A' && r <= 'Z') || r == '-'
}

func isLowerOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// SessionCode is the code of Cavea for a session: "cag_5/27-10-2026/22:15". It names
// the hall and the start, in Tbilisi. Two codes of the same text are equal with ==.
type SessionCode struct {
	code    string
	hallKey string
	// start is in Unix seconds, so that the zone of a time.Time does not enter ==.
	start int64
}

// ParseSessionCode returns the session code of a text <hall key>/DD-MM-YYYY/HH:MM. The
// hall key is two parts of letters and digits with a _ between them.
func ParseSessionCode(text string) (SessionCode, error) {
	hallKey, when, found := strings.Cut(text, "/")
	if !found || !isHallKey(hallKey) {
		return SessionCode{}, fmt.Errorf("%w: %q", ErrInvalidSessionCode, text)
	}

	start, err := time.ParseInLocation(sessionLayout, when, kernel.TbilisiZone())
	if err != nil || start.Format(sessionLayout) != when {
		return SessionCode{}, fmt.Errorf("%w: %q", ErrInvalidSessionCode, text)
	}

	return SessionCode{code: text, hallKey: hallKey, start: start.Unix()}, nil
}

func isHallKey(text string) bool {
	cinema, hall, found := strings.Cut(text, "_")

	return found && isAlphanumeric(cinema) && isAlphanumeric(hall)
}

func isAlphanumeric(text string) bool {
	if text == "" {
		return false
	}

	for _, r := range text {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}

	return true
}

// String returns the code: "cag_5/27-10-2026/22:15".
func (c SessionCode) String() string {
	return c.code
}

// HallKey returns the key of the hall of the session: "cag_5".
func (c SessionCode) HallKey() string {
	return c.hallKey
}

// Start returns the start of the session, in Tbilisi.
func (c SessionCode) Start() time.Time {
	return time.Unix(c.start, 0).In(kernel.TbilisiZone())
}

// Date returns the date of the start of the session, in Tbilisi.
func (c SessionCode) Date() kernel.Date {
	return kernel.Today(c.Start())
}
