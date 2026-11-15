package domain

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// MaxTextLength is the count of characters of the longest search text.
const MaxTextLength = 128

// MaxPageSize is the count of items of the largest page. The Parliament gives no page of
// more than 50 bills.
const MaxPageSize = 50

// SearchText is a text of 1 to MaxTextLength characters, with no space at the ends.
type SearchText struct {
	text string
}

// NewSearchText trims the text and checks its length.
func NewSearchText(text string) (SearchText, error) {
	trimmed := strings.TrimSpace(text)

	if count := utf8.RuneCountInString(trimmed); count == 0 || count > MaxTextLength {
		return SearchText{}, fmt.Errorf("%w: %d characters", ErrInvalidText, count)
	}

	return SearchText{text: trimmed}, nil
}

// String returns the text.
func (t SearchText) String() string {
	return t.text
}

// IsZero reports whether the text is absent.
func (t SearchText) IsZero() bool {
	return t.text == ""
}

// HasGeorgian reports whether the text holds a letter of Mkhedruli, Asomtavruli, or
// Mtavruli.
func (t SearchText) HasGeorgian() bool {
	return strings.ContainsFunc(t.text, func(r rune) bool {
		return unicode.In(r, unicode.Georgian) && unicode.IsLetter(r)
	})
}

// ID is the positive identifier of a bill or a vote at the Parliament.
type ID int

// NewID checks that the value is positive.
func NewID(value int) (ID, error) {
	if value < 1 {
		return 0, fmt.Errorf("%w: %d", ErrInvalidID, value)
	}

	return ID(value), nil
}

// Period is a range of days, both ends included. Each end is optional.
type Period struct {
	From, To *kernel.Date
}

// NewPeriod checks that the period starts no later than it ends.
func NewPeriod(from, to *kernel.Date) (Period, error) {
	if from != nil && to != nil && from.After(*to) {
		return Period{}, fmt.Errorf("%w: %s, %s", ErrInvertedRange, from, to)
	}

	return Period{From: from, To: to}, nil
}

// IsZero reports whether the period has no end.
func (p Period) IsZero() bool {
	return p.From == nil && p.To == nil
}

// Page is a page of results: Number from 1, Size from 1 to MaxPageSize. The tool checks
// both.
type Page struct {
	Number, Size int
}

// Start returns the count of the items before the page.
func (p Page) Start() int {
	return (p.Number - 1) * p.Size
}
