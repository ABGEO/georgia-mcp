package app

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	minQueryRunes = 3
	maxQueryRunes = 100
)

// CodeQuery asks the postal codes of the addresses that match a text.
type CodeQuery struct {
	Text     string
	Language kernel.Language
	// Limit is the maximum count of addresses. The tool gives the default.
	Limit int
}

// Codes is the addresses of a search with their postal codes.
type Codes struct {
	// Addresses are the first Limit addresses, in the order of Georgian Post.
	Addresses []domain.Address
	// Total is the count of addresses that Georgian Post matched.
	Total  int
	ReadAt time.Time
	Stale  bool
}

// SearchCodes finds the postal codes of the addresses that match a text.
type SearchCodes struct {
	source PostalCodeSource
}

// NewSearchCodes returns the use case.
func NewSearchCodes(source PostalCodeSource) *SearchCodes {
	return &SearchCodes{source: source}
}

// Search returns the addresses up to the limit, and the count of all the matches.
func (s *SearchCodes) Search(ctx context.Context, query CodeQuery) (Codes, error) {
	text := strings.Join(strings.Fields(query.Text), " ")
	if count := utf8.RuneCountInString(text); count < minQueryRunes || count > maxQueryRunes {
		return Codes{}, fmt.Errorf("%w: %d characters", ErrInvalidQuery, count)
	}

	list, err := s.source.Addresses(ctx, text, query.Language)
	if err != nil {
		return Codes{}, fmt.Errorf("searching the postal codes: %w", err)
	}

	found := list.Addresses
	if query.Limit > 0 && len(found) > query.Limit {
		found = found[:query.Limit]
	}

	return Codes{
		Addresses: found,
		Total:     len(list.Addresses),
		ReadAt:    list.ReadAt,
		Stale:     list.Stale,
	}, nil
}
