package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
)

// SearchFilter holds a code, a period, or both. The zero value of each is absent.
type SearchFilter struct {
	Code   domain.CadastralCode
	Period domain.Period
}

// IsZero reports whether the filter has no code and no period.
func (f SearchFilter) IsZero() bool {
	return f.Code.IsZero() && f.Period.IsZero()
}

// MatchRead is the applications that match a filter, in the order of the source.
type MatchRead struct {
	// Applications holds at most domain.MaxMatches applications, with no decision text.
	Applications []domain.Application
	// Total is the count of all the matches.
	Total  int
	ReadAt time.Time
	Stale  bool
}

// ApplicationRead is one application with its decision text.
type ApplicationRead struct {
	Application domain.Application
	ReadAt      time.Time
	Stale       bool
}

// ApplicationSource gives the applications of TAS.
type ApplicationSource interface {
	// Match reads the applications that match the filter.
	Match(ctx context.Context, filter SearchFilter) (MatchRead, error)
	// Application reads the application with the number, or ErrApplicationNotFound.
	Application(ctx context.Context, number domain.ApplicationNumber) (ApplicationRead, error)
}
