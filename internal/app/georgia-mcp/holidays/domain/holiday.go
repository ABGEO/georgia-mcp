package domain

import (
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// HolidayName is the name of a public holiday in Georgian and in English.
type HolidayName struct {
	georgian, english string
}

// NewHolidayName returns the name of the two texts, trimmed. It fails when both are empty.
func NewHolidayName(georgian, english string) (HolidayName, error) {
	name := HolidayName{georgian: strings.TrimSpace(georgian), english: strings.TrimSpace(english)}
	if name.georgian == "" && name.english == "" {
		return HolidayName{}, ErrInvalidHolidayName
	}

	return name, nil
}

// In returns the name in the language, or the name in the other language when it is empty.
func (n HolidayName) In(language kernel.Language) string {
	first, second := n.english, n.georgian
	if language == kernel.LanguageGeorgian {
		first, second = second, first
	}

	if first != "" {
		return first
	}

	return second
}

// Holiday is one public holiday of Georgia.
type Holiday struct {
	Date kernel.Date
	Name HolidayName
}

// Weekday returns the day of the week of the date.
func (h Holiday) Weekday() time.Weekday {
	return h.Date.Time().Weekday()
}
