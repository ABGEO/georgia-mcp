package domain

import (
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Period is the offer period: the dates from the start date to the end date, both
// included. A zero date is a date that the bank does not publish.
type Period struct {
	Start kernel.Date
	End   kernel.Date
}

// NewPeriod returns an offer period. It refuses an end before the start.
func NewPeriod(start, end kernel.Date) (Period, error) {
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return Period{}, fmt.Errorf("%w: %s to %s", ErrInvalidPeriod, start, end)
	}

	return Period{Start: start, End: end}, nil
}

// Overlaps reports whether the period shares a date with the range from one date to
// another. A zero start is open to the past, and a zero end is open to the future.
func (p Period) Overlaps(from, to kernel.Date) bool {
	startsInTime := p.Start.IsZero() || !p.Start.After(to)
	endsInTime := p.End.IsZero() || !p.End.Before(from)

	return startsInTime && endsInTime
}

// EndsBefore reports whether the period ends before a date. A zero end never does.
func (p Period) EndsBefore(day kernel.Date) bool {
	return !p.End.IsZero() && p.End.Before(day)
}
