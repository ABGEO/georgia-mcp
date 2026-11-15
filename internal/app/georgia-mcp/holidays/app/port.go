package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/domain"
)

// Calendar is the public holidays of one year, by date, and the instant of the read.
type Calendar struct {
	Year     int
	Holidays []domain.Holiday
	ReadAt   time.Time
	Stale    bool
}

// HolidaySource gives the public holidays of Georgia.
type HolidaySource interface {
	// Holidays returns the calendar of the year.
	Holidays(ctx context.Context, year int) (Calendar, error)
}
