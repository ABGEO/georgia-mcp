package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	defaultLimit = 10
	maxRangeDays = 366
)

// SearchQuery asks for the offers that match each of its filters. An empty list asks
// every value. A value named twice counts once.
type SearchQuery struct {
	Banks      []domain.BankCode
	Tiers      []domain.Tier
	Categories []domain.Category
	Kinds      []domain.BenefitKind
	Cards      []domain.CardCode
	// From is the first date of the range. The zero date is today in Tbilisi.
	From kernel.Date
	// To is the last date of the range. The zero date is From.
	To kernel.Date
	// Text is empty for no text filter.
	Text     string
	Language kernel.Language
	// Limit is the maximum count of offers of each bank. Zero gives 10.
	Limit int
	// Offset is the count of matching offers of each bank to skip.
	Offset int
}

// BankOffers is the answer of one bank: at most Limit offers after Offset, and the count
// of every matching offer. NextOffset is the offset of the next page, or 0 after the last.
type BankOffers struct {
	Bank       domain.Bank
	Offers     []domain.Offer
	Total      int
	NextOffset int
	ReadAt     time.Time
	Stale      bool
}

// SearchResult holds the answer of each bank that answered, sorted by code, and the
// banks that failed with no offers held, sorted by code.
type SearchResult struct {
	Banks       []BankOffers
	Unavailable []domain.Bank
}

// SearchOffers finds the current offers of the banks of the server.
type SearchOffers struct {
	banks banks
	now   func() time.Time
}

// NewSearchOffers returns the use case over one source for each bank.
func NewSearchOffers(sources []OfferSource, now func() time.Time) *SearchOffers {
	return &SearchOffers{banks: newBanks(sources), now: now}
}

// Banks returns the banks of the server, sorted by code.
func (u *SearchOffers) Banks() []domain.Bank {
	return slices.Clone(u.banks.list)
}

// Cards returns the cards that each bank of the server knows.
func (u *SearchOffers) Cards() map[domain.BankCode][]domain.Card {
	cards := make(map[domain.BankCode][]domain.Card, len(u.banks.list))
	for code, source := range u.banks.sources {
		cards[code] = source.Cards()
	}

	return cards
}

// Search asks each requested bank at once and waits for all. A bank that fails joins
// Unavailable. When every requested bank fails, Search returns an error that wraps
// kernel.ErrSourceUnavailable.
func (u *SearchOffers) Search(ctx context.Context, query SearchQuery) (SearchResult, error) {
	today := kernel.Today(u.now())

	query, err := withRange(query, today)
	if err != nil {
		return SearchResult{}, err
	}

	requested, err := u.banks.resolve(query.Banks)
	if err != nil {
		return SearchResult{}, err
	}

	catalogs := make([]Catalog, len(requested))
	errs := make([]error, len(requested))

	var wg sync.WaitGroup

	for i, bank := range requested {
		wg.Go(func() {
			catalogs[i], errs[i] = u.banks.sources[bank.Code].Offers(ctx, query.Language)
		})
	}

	wg.Wait()

	var result SearchResult

	for i, bank := range requested {
		if errs[i] != nil {
			result.Unavailable = append(result.Unavailable, bank)

			continue
		}

		result.Banks = append(result.Banks, match(bank, catalogs[i], query, today))
	}

	if len(result.Banks) == 0 {
		return SearchResult{}, fmt.Errorf(
			"%w: %s: %w",
			kernel.ErrSourceUnavailable,
			bankNames(requested),
			errors.Join(errs...),
		)
	}

	return result, nil
}

// withRange sets the default dates and the default limit, and checks the range.
func withRange(query SearchQuery, today kernel.Date) (SearchQuery, error) {
	if query.From.IsZero() {
		query.From = today
	}

	if query.To.IsZero() {
		query.To = query.From
	}

	if query.To.Before(query.From) || kernel.DaysBetween(query.From, query.To) > maxRangeDays {
		return SearchQuery{}, fmt.Errorf("%w: %s to %s", ErrInvalidRange, query.From, query.To)
	}

	if query.Limit < 1 {
		query.Limit = defaultLimit
	}

	query.Text = strings.ToLower(query.Text)

	return query, nil
}

// match returns the matching offers of a catalog, sorted and cut to the limit.
func match(bank domain.Bank, catalog Catalog, query SearchQuery, today kernel.Date) BankOffers {
	answer := BankOffers{Bank: bank, ReadAt: catalog.ReadAt, Stale: catalog.Stale}

	for _, offer := range catalog.Offers {
		if !offer.Period.EndsBefore(today) && matches(offer, query) {
			answer.Offers = append(answer.Offers, offer)
		}
	}

	slices.SortStableFunc(answer.Offers, compareOffers)

	answer.Total = len(answer.Offers)
	first := min(max(query.Offset, 0), answer.Total)
	end := min(first+query.Limit, answer.Total)
	answer.Offers = answer.Offers[first:end]

	if end < answer.Total {
		answer.NextOffset = end
	}

	return answer
}

func matches(offer domain.Offer, query SearchQuery) bool {
	return offer.Period.Overlaps(query.From, query.To) &&
		sharesOne(offer.Tiers, query.Tiers) &&
		sharesOne(offer.Categories, query.Categories) &&
		sharesOne(offer.Kinds, query.Kinds) &&
		matchesCards(offer.Cards, query.Cards) &&
		holdsText(offer, query.Text)
}

// sharesOne reports whether a filter is empty or shares one value with the offer.
func sharesOne[V comparable](values, filter []V) bool {
	return len(filter) == 0 || slices.ContainsFunc(values, func(value V) bool {
		return slices.Contains(filter, value)
	})
}

// matchesCards reports whether a card filter matches. An offer with no card is for
// every card of its tiers.
func matchesCards(cards []domain.Card, filter []domain.CardCode) bool {
	if len(filter) == 0 || len(cards) == 0 {
		return true
	}

	return slices.ContainsFunc(cards, func(card domain.Card) bool {
		return slices.Contains(filter, card.Code)
	})
}

func holdsText(offer domain.Offer, text string) bool {
	if text == "" {
		return true
	}

	fields := append([]string{offer.Title, offer.Partner, offer.Benefit}, offer.BankCategories...)

	return slices.ContainsFunc(fields, func(field string) bool {
		return strings.Contains(strings.ToLower(field), text)
	})
}

// compareOffers sorts by end date, the open offers last, then by title, then by id.
func compareOffers(left, right domain.Offer) int {
	leftEnd, rightEnd := left.Period.End, right.Period.End

	switch {
	case leftEnd.IsZero() && !rightEnd.IsZero():
		return 1
	case !leftEnd.IsZero() && rightEnd.IsZero():
		return -1
	case leftEnd.Before(rightEnd):
		return -1
	case leftEnd.After(rightEnd):
		return 1
	}

	return cmp.Or(cmp.Compare(left.Title, right.Title), cmp.Compare(left.ID, right.ID))
}
