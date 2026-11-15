package app

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
)

// SearchQuery is a filter and the most applications to return.
type SearchQuery struct {
	Filter SearchFilter
	Limit  int
}

// ApplicationList is the newest applications of a search, and the count of all matches.
type ApplicationList struct {
	Applications []domain.Application
	Total        int
	ReadAt       time.Time
	Stale        bool
}

// SearchApplications returns the newest applications for a code, a period, or both.
type SearchApplications struct {
	source ApplicationSource
}

// NewSearchApplications returns the use case over the source.
func NewSearchApplications(source ApplicationSource) *SearchApplications {
	return &SearchApplications{source: source}
}

// Search reads the matches, refuses more than domain.MaxMatches, sorts them newest first,
// and takes the first Limit.
func (u *SearchApplications) Search(
	ctx context.Context,
	query SearchQuery,
) (ApplicationList, error) {
	if query.Filter.IsZero() {
		return ApplicationList{}, ErrNoFilter
	}

	read, err := u.source.Match(ctx, query.Filter)
	if err != nil {
		return ApplicationList{}, fmt.Errorf("searching the applications: %w", err)
	}

	if read.Total > domain.MaxMatches {
		return ApplicationList{}, &TooManyMatchesError{Count: read.Total}
	}

	sorted := domain.SortApplications(read.Applications)
	if len(sorted) > query.Limit {
		sorted = sorted[:query.Limit]
	}

	return ApplicationList{
		Applications: sorted,
		Total:        read.Total,
		ReadAt:       read.ReadAt,
		Stale:        read.Stale,
	}, nil
}
