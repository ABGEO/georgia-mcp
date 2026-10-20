package app_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func search(t *testing.T, sources []app.OfferSource, query app.SearchQuery) []string {
	t.Helper()

	if query.Limit == 0 {
		query.Limit = 30
	}

	result, err := app.NewSearchOffers(sources, now).Search(t.Context(), query)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	got := make([]string, 0, len(result.Banks))
	for _, bank := range result.Banks {
		got = append(got, ids(bank.Offers)...)
	}

	return got
}

// BOGOFFER-SC-011: The search leaves out an ended offer, and keeps the offers whose
// period overlaps the requested dates. The default range is today.
func TestSearchFiltersThePeriod(t *testing.T) {
	t.Parallel()

	source := newFake(bog,
		offer(t, "A", period(t, "", "2026-10-19")),
		offer(t, "B", period(t, "2026-10-20", "2026-10-31")),
		offer(t, "C", period(t, "2026-10-30", "")),
		offer(t, "D"),
	)
	sources := []app.OfferSource{source}

	if diff := cmp.Diff([]string{"B", "D"}, search(t, sources, app.SearchQuery{})); diff != "" {
		t.Errorf("today mismatch (-want +got):\n%s", diff)
	}

	october := app.SearchQuery{From: mustDate(t, "2026-10-01"), To: mustDate(t, "2026-10-31")}
	if diff := cmp.Diff([]string{"B", "C", "D"}, search(t, sources, october)); diff != "" {
		t.Errorf("october mismatch (-want +got):\n%s", diff)
	}
}

// BOGOFFER-SC-012: Each filter keeps the offers with one of its values. An offer that
// names no card matches every card.
func TestSearchFiltersTheValues(t *testing.T) {
	t.Parallel()

	source := newFake(bog,
		offer(t, "E", func(o *domain.Offer) {
			o.Tiers = []domain.Tier{domain.TierPremium}
			o.Categories = []domain.Category{domain.CategoryLeisure}
			o.Cards = []domain.Card{{Code: "solo", Name: "SOLO"}}
		}),
		offer(t, "F", func(o *domain.Offer) {
			o.Categories = []domain.Category{domain.CategoryLeisure}
		}),
		offer(t, "G", func(o *domain.Offer) {
			o.Tiers = []domain.Tier{domain.TierPremium}
			o.Categories = []domain.Category{domain.CategoryDining}
			o.Cards = []domain.Card{{Code: "amex", Name: "Amex"}}
		}),
	)
	sources := []app.OfferSource{source}

	premium := app.SearchQuery{
		Tiers:      []domain.Tier{domain.TierPremium},
		Categories: []domain.Category{domain.CategoryLeisure},
		Cards:      []domain.CardCode{"solo"},
	}
	if diff := cmp.Diff([]string{"E"}, search(t, sources, premium)); diff != "" {
		t.Errorf("premium mismatch (-want +got):\n%s", diff)
	}

	visa := app.SearchQuery{Cards: []domain.CardCode{"visa"}}
	if diff := cmp.Diff([]string{"F"}, search(t, sources, visa)); diff != "" {
		t.Errorf("visa mismatch (-want +got):\n%s", diff)
	}

	kinds := app.SearchQuery{Kinds: []domain.BenefitKind{domain.KindCashback}}
	if got := search(t, sources, kinds); len(got) != 0 {
		t.Errorf("cashback = %v, want none", got)
	}
}

// BOGOFFER-SC-013: The text filter finds the text in the title with no regard to case.
func TestSearchFiltersTheText(t *testing.T) {
	t.Parallel()

	source := newFake(bog,
		offer(t, "17409", func(o *domain.Offer) { o.Title = "Ronny's Pizza" }),
		offer(t, "16902", func(o *domain.Offer) {
			o.Title = "Mazda"
			o.Benefit = "Special Offer"
		}),
	)

	got := search(t, []app.OfferSource{source}, app.SearchQuery{Text: "PIZZA"})
	if diff := cmp.Diff([]string{"17409"}, got); diff != "" {
		t.Errorf("Search() mismatch (-want +got):\n%s", diff)
	}
}

// BOGOFFER-SC-014: The search cuts each bank to the limit, sorts by end date then
// title, and counts every match.
func TestSearchCutsToTheLimit(t *testing.T) {
	t.Parallel()

	offers := make([]domain.Offer, 0, 25)

	for i := 25; i > 0; i-- {
		id := fmt.Sprintf("%02d", i)
		offers = append(offers, offer(t, id, period(t, "", fmt.Sprintf("2026-11-%02d", i))))
	}

	offers = append(offers, offer(t, "open"), offer(t, "early", func(o *domain.Offer) {
		o.Title = "A early"
		o.Period.End = mustDate(t, "2026-11-01")
	}))

	result, err := app.NewSearchOffers([]app.OfferSource{newFake(bog, offers...)}, now).
		Search(t.Context(), app.SearchQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	bank := result.Banks[0]
	if bank.Total != 27 || !bank.ReadAt.Equal(readAt()) {
		t.Errorf("Total = %d, ReadAt = %v; want 27, %v", bank.Total, bank.ReadAt, readAt())
	}

	want := []string{"early", "01", "02", "03", "04", "05", "06", "07", "08", "09"}
	if diff := cmp.Diff(want, ids(bank.Offers)); diff != "" {
		t.Errorf("offers mismatch (-want +got):\n%s", diff)
	}
}

// BOGOFFER-SC-016: A failed bank joins Unavailable. When every requested bank fails,
// the search fails. An unknown bank code is refused with the known codes.
func TestSearchReportsTheBanks(t *testing.T) {
	t.Parallel()

	failing := newFake(xyz)
	failing.err = fmt.Errorf("reading: %w", kernel.ErrSourceUnavailable)

	use := app.NewSearchOffers([]app.OfferSource{failing, newFake(bog, offer(t, "1"))}, now)

	result, err := use.Search(t.Context(), app.SearchQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(result.Banks) != 1 || result.Banks[0].Bank.Code != bog ||
		len(result.Unavailable) != 1 || result.Unavailable[0].Code != xyz {
		t.Errorf("Search() = %+v, want bog and unavailable xyz", result)
	}

	_, err = use.Search(t.Context(), app.SearchQuery{Banks: []domain.BankCode{xyz}, Limit: 10})
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Search(xyz) error = %v, want ErrSourceUnavailable", err)
	}

	_, err = use.Search(t.Context(), app.SearchQuery{Banks: []domain.BankCode{"abc"}, Limit: 10})

	unknown, ok := errors.AsType[*app.UnknownBankError](err)
	if !ok || !errors.Is(err, app.ErrUnknownBank) {
		t.Fatalf("Search(abc) error = %v, want *UnknownBankError", err)
	}

	if diff := cmp.Diff([]domain.BankCode{bog, xyz}, unknown.Known); diff != "" {
		t.Errorf("Known mismatch (-want +got):\n%s", diff)
	}
}

// BOGOFFER-SC-017: The search refuses an end before the start, and more than 366 days.
func TestSearchRefusesTheRange(t *testing.T) {
	t.Parallel()

	use := app.NewSearchOffers([]app.OfferSource{newFake(bog)}, now)

	ranges := [][2]string{{"2026-10-31", "2026-10-01"}, {"2026-01-01", "2027-01-03"}}
	for _, dates := range ranges {
		query := app.SearchQuery{From: mustDate(t, dates[0]), To: mustDate(t, dates[1]), Limit: 10}
		if _, err := use.Search(t.Context(), query); !errors.Is(err, app.ErrInvalidRange) {
			t.Errorf("Search(%v) error = %v, want ErrInvalidRange", dates, err)
		}
	}

	query := app.SearchQuery{
		From:  mustDate(t, "2026-01-01"),
		To:    mustDate(t, "2027-01-02"),
		Limit: 10,
	}
	if _, err := use.Search(t.Context(), query); err != nil {
		t.Errorf("Search(366 days) error = %v, want nil", err)
	}
}

// BOGOFFER-SC-023: The offset skips the matching offers of each bank. The pages share no
// offer, keep the order of the answer, and give the next offset while more remain.
func TestSearchPagesWithTheOffset(t *testing.T) {
	t.Parallel()

	offers := make([]domain.Offer, 0, 25)
	for i := range 25 {
		offers = append(
			offers,
			offer(t, fmt.Sprintf("%02d", i), period(t, "", fmt.Sprintf("2026-11-%02d", 25-i))),
		)
	}

	use := app.NewSearchOffers([]app.OfferSource{newFake(bog, offers...)}, now)

	whole, err := use.Search(t.Context(), app.SearchQuery{Limit: 30})
	if err != nil {
		t.Fatalf("Search(limit 30) error = %v", err)
	}

	paged := make([]string, 0, len(offers))

	for _, page := range []struct{ offset, count, next int }{
		{offset: 0, count: 10, next: 10},
		{offset: 10, count: 10, next: 20},
		{offset: 20, count: 5, next: 0},
		{offset: 30, count: 0, next: 0},
	} {
		result, err := use.Search(t.Context(), app.SearchQuery{Limit: 10, Offset: page.offset})
		if err != nil {
			t.Fatalf("Search(offset %d) error = %v", page.offset, err)
		}

		bank := result.Banks[0]
		if len(bank.Offers) != page.count || bank.NextOffset != page.next || bank.Total != 25 {
			t.Errorf("offset %d = %d offers, next %d, total %d; want %d, %d, 25",
				page.offset, len(bank.Offers), bank.NextOffset, bank.Total, page.count, page.next)
		}

		paged = append(paged, ids(bank.Offers)...)
	}

	if diff := cmp.Diff(ids(whole.Banks[0].Offers), paged); diff != "" {
		t.Errorf("pages mismatch (-whole +pages):\n%s", diff)
	}
}
