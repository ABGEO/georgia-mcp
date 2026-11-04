package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const maxQueryRunes = 100

// TableQuery asks the tables whose title, folder, or code holds every word of a text.
type TableQuery struct {
	Text     string
	Language kernel.Language
	// Limit is the maximum count of tables. The tool gives the default. 0 gives every
	// match.
	Limit int
}

// TableMatches is the tables of a search.
type TableMatches struct {
	// Entries are the first Limit matches, the newest update first.
	Entries []domain.CatalogEntry
	// Total is the count of every match.
	Total  int
	ReadAt time.Time
	Stale  bool
}

// SearchTables finds the tables of the catalog that match a text.
type SearchTables struct {
	source CatalogSource
}

// NewSearchTables returns the use case.
func NewSearchTables(source CatalogSource) *SearchTables {
	return &SearchTables{source: source}
}

// Search returns the matches up to the limit, and the count of every match.
func (s *SearchTables) Search(ctx context.Context, query TableQuery) (TableMatches, error) {
	text := strings.TrimSpace(query.Text)
	if count := utf8.RuneCountInString(text); count == 0 || count > maxQueryRunes {
		return TableMatches{}, fmt.Errorf("%w: %d characters", ErrInvalidQuery, count)
	}

	catalog, err := s.source.Catalog(ctx, query.Language)
	if err != nil {
		return TableMatches{}, fmt.Errorf("reading the catalog: %w", err)
	}

	words := strings.Fields(strings.ToLower(text))
	matches := []domain.CatalogEntry{}

	for _, entry := range catalog.Entries {
		if matchesEvery(entry, words) {
			matches = append(matches, entry)
		}
	}

	slices.SortStableFunc(matches, func(a, b domain.CatalogEntry) int {
		if order := b.Updated.Compare(a.Updated); order != 0 {
			return order
		}

		return cmp.Compare(a.Code.String(), b.Code.String())
	})

	total := len(matches)
	if query.Limit > 0 && total > query.Limit {
		matches = matches[:query.Limit]
	}

	return TableMatches{
		Entries: matches, Total: total, ReadAt: catalog.ReadAt, Stale: catalog.Stale,
	}, nil
}

func matchesEvery(entry domain.CatalogEntry, words []string) bool {
	text := strings.ToLower(entry.Title + "\n" + entry.Folder + "\n" + entry.Code.String())

	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}

	return true
}
