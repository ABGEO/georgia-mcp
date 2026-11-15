package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	maxExactMatches = 3
	quotes          = "„“”\"«»"
)

// SearchResult is one page of acts, with the exact title matches first.
type SearchResult struct {
	Acts   []domain.ActSummary
	Total  int
	ReadAt time.Time
	Stale  bool
}

// SearchActs searches the acts.
type SearchActs struct {
	source ActSource
	logger *slog.Logger
}

// NewSearchActs returns the use case. The logger names the vendor of the source.
func NewSearchActs(source ActSource, logger *slog.Logger) *SearchActs {
	return &SearchActs{source: source, logger: logger}
}

// Search returns at most limit acts of one page. A Georgian query that equals the title of
// an active act puts that act first, when the query asks the first page of titles with no
// filter.
func (s *SearchActs) Search(
	ctx context.Context,
	query domain.SearchQuery,
	limit int,
) (SearchResult, error) {
	if !exactTitleRule(query) {
		page, err := s.source.Search(ctx, query)
		if err != nil {
			return SearchResult{}, fmt.Errorf("searching the acts: %w", err)
		}

		return toResult(page, nil, limit), nil
	}

	var (
		page    SearchPage
		titles  TitleList
		pageErr error
		titlErr error
		group   sync.WaitGroup
	)

	group.Go(func() { page, pageErr = s.source.Search(ctx, query) })
	group.Go(func() { titles, titlErr = s.source.Titles(ctx, query.Text) })
	group.Wait()

	if pageErr != nil {
		return SearchResult{}, fmt.Errorf("searching the acts: %w", pageErr)
	}

	// A failure of the titles leaves the page of the source.
	var matches []matchedAct
	if titlErr == nil {
		matches = s.readMatches(ctx, exactMatches(titles.Titles, query.Text), query.Language)
	} else {
		logLeftOut(ctx, s.logger, titlErr)
	}

	result := toResult(page, matches, limit)
	result.Stale = result.Stale || (titlErr == nil && titles.Stale)

	return result, nil
}

// logLeftOut records a read that the answer leaves out. The tool call succeeds, so its
// own record holds no failure.
func logLeftOut(ctx context.Context, logger *slog.Logger, err error) {
	logger.WarnContext(ctx, "source unavailable", slog.Any("error", err))
}

func exactTitleRule(query domain.SearchQuery) bool {
	return !query.InText && query.Page <= 1 && !query.HasFilter() &&
		strings.ContainsFunc(query.Text, func(char rune) bool {
			return unicode.Is(unicode.Georgian, char)
		})
}

// exactMatches returns at most 3 active titles equal to the text.
func exactMatches(titles []domain.ActSummary, text string) []domain.ActID {
	want := normalizeTitle(text)

	var ids []domain.ActID

	for _, title := range titles {
		if title.Status == domain.StatusActive && normalizeTitle(title.Title) == want {
			ids = append(ids, title.ID)
		}

		if len(ids) == maxExactMatches {
			break
		}
	}

	return ids
}

func normalizeTitle(text string) string {
	cleaned := strings.Map(func(char rune) rune {
		if strings.ContainsRune(quotes, char) {
			return -1
		}

		return char
	}, text)

	return strings.Join(strings.Fields(cleaned), " ")
}

// readMatches reads each matching act in parallel and returns those it read, in order. An
// act with no English text is read in Georgian.
func (s *SearchActs) readMatches(
	ctx context.Context,
	ids []domain.ActID,
	language kernel.Language,
) []matchedAct {
	read := make([]matchedAct, len(ids))

	var group sync.WaitGroup

	for i, id := range ids {
		group.Go(func() {
			text, err := s.source.Act(ctx, id, nil, language)
			if errors.Is(err, ErrNoTranslation) {
				text, err = s.source.Act(ctx, id, nil, kernel.LanguageGeorgian)
			}

			if err != nil {
				logLeftOut(ctx, s.logger, err)

				return
			}

			read[i] = matchedAct{text: text, ok: true}
		})
	}

	group.Wait()

	matches := make([]matchedAct, 0, len(read))

	for _, match := range read {
		if match.ok {
			matches = append(matches, match)
		}
	}

	return matches
}

type matchedAct struct {
	text ActText
	ok   bool
}

func toResult(page SearchPage, matches []matchedAct, limit int) SearchResult {
	acts := make([]domain.ActSummary, 0, len(page.Acts)+len(matches))
	seen := map[domain.ActID]bool{}
	stale := page.Stale

	for _, match := range matches {
		act := match.text.Act
		acts = append(acts, domain.ActSummary{
			ID: act.ID, Title: act.Title, Type: act.Type, Issuer: act.Issuer,
			Number: act.Number, Signed: act.Signed, Status: domain.StatusActive,
		})
		seen[act.ID] = true
		stale = stale || match.text.Stale
	}

	for _, act := range page.Acts {
		if !seen[act.ID] {
			acts = append(acts, act)
		}
	}

	if limit > 0 && len(acts) > limit {
		acts = acts[:limit]
	}

	return SearchResult{Acts: acts, Total: page.Total, ReadAt: page.ReadAt, Stale: stale}
}
