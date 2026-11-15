package domain

import (
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// MaxPeriodDays is the longest period of a search, the first and the last day included.
const MaxPeriodDays = 31

// Period is the days of registration of a search, both ends included.
type Period struct {
	From, To kernel.Date
}

// NewPeriod checks that the period starts no later than it ends and holds at most
// MaxPeriodDays days.
func NewPeriod(from, to kernel.Date) (Period, error) {
	if from.After(to) {
		return Period{}, fmt.Errorf("%w: %s, %s", ErrInvalidPeriod, from, to)
	}

	if days := kernel.DaysBetween(from, to) + 1; days > MaxPeriodDays {
		return Period{}, fmt.Errorf("%w: %d days", ErrPeriodTooLong, days)
	}

	return Period{From: from, To: to}, nil
}

// IsZero reports whether the period is absent.
func (p Period) IsZero() bool {
	return p.From.IsZero()
}
