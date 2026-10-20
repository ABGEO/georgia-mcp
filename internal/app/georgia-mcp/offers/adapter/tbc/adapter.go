// Package tbc reads the card offers from the client of TBC Bank and converts its answer
// to the domain. It is the anti-corruption layer of the context for TBC.
package tbc

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
)

const (
	pageSize = 1000
	// maxPages bounds a list. TBC listed 518 offers in one page on 2026-10-20.
	maxPages = 5
	// parallelLists bounds the filter lists that run at once.
	parallelLists = 4
	dateLength    = len("2026-10-20")

	conceptURL = "https://tbcconcept.ge/%s/offers/%s"
	bankURL    = "https://tbcbank.ge/%s/offers/all-offers/%s"
)

// filter is one filter of TBC: its group, its code, and its name in the locale.
type filter struct {
	group, code, name string
}

func (f filter) key() string {
	return f.group + ":" + f.code
}

// Adapter implements app.OfferSource with the client of TBC Bank.
type Adapter struct {
	client     *tbcoffers.Client
	now        func() time.Time
	logger     *slog.Logger
	warned     sync.Map
	tiers      map[string]domain.Tier
	categories map[string]domain.Category
	kinds      map[string][]domain.BenefitKind
	cards      map[string]domain.Card
}

var _ app.OfferSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each code that no table of the adapter holds,
// and one for each offer that it leaves out.
func NewAdapter(client *tbcoffers.Client, now func() time.Time, logger *slog.Logger) *Adapter {
	return &Adapter{
		client:     client,
		now:        now,
		logger:     logger,
		tiers:      tierTable(),
		categories: categoryTable(),
		kinds:      kindTable(),
		cards:      cardTable(),
	}
}

// Bank returns TBC Bank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "tbc", Name: "TBC Bank"}
}

// Cards returns the cards of the table of the adapter, sorted by code.
func (a *Adapter) Cards() []domain.Card {
	return slices.SortedFunc(maps.Values(a.cards), func(left, right domain.Card) int {
		return cmp.Compare(left.Code, right.Code)
	})
}

// Offers reads the filters of each segment, every offer, and the offers of each filter,
// and gives each offer the values of the filters that list it. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) Offers(ctx context.Context, language kernel.Language) (app.Catalog, error) {
	locale := localeOf(language)

	filters, err := a.filters(ctx, locale)
	if err != nil {
		return app.Catalog{}, unavailable(err)
	}

	entries, err := a.list(ctx, locale, "")
	if err != nil {
		return app.Catalog{}, unavailable(err)
	}

	members, err := a.members(ctx, locale, filters)
	if err != nil {
		return app.Catalog{}, unavailable(err)
	}

	catalog := app.Catalog{Offers: make([]domain.Offer, 0, len(entries)), ReadAt: a.now()}

	for _, entry := range entries {
		offer, err := a.toOffer(ctx, entry, members[entry.ID], language)
		if err != nil {
			a.logger.WarnContext(ctx, "TBC lists an offer that the server leaves out",
				slog.String("id", entry.ID), slog.String("cause", err.Error()))

			continue
		}

		catalog.Offers = append(catalog.Offers, offer)
	}

	return catalog, nil
}

// Terms reads the page of the offer. The slug is the last segment of its address.
func (a *Adapter) Terms(
	ctx context.Context,
	offer domain.Offer,
	language kernel.Language,
) (app.Terms, error) {
	address, err := url.Parse(offer.URL)
	if err != nil {
		return app.Terms{}, unavailable(fmt.Errorf("reading the address %q: %w", offer.URL, err))
	}

	entry, err := a.client.Offer(ctx, path.Base(address.Path), localeOf(language))
	if err != nil {
		return app.Terms{}, unavailable(err)
	}

	return app.Terms{Text: treeText(entry.Content), ReadAt: a.now()}, nil
}

// filters returns the first level of the groups of the filters of each segment, joined
// by group and code, in the order of the answers.
func (a *Adapter) filters(ctx context.Context, locale string) ([]filter, error) {
	var joined []filter

	seen := map[string]bool{}
	groups := []string{groupCategory, groupProductType, groupOfferType, groupCardType}

	for _, segment := range []string{"Concept", "ForEveryone", "ForYouth"} {
		tree, err := a.client.Filters(ctx, segment, locale)
		if err != nil {
			return nil, fmt.Errorf("reading the filters of TBC: %w", err)
		}

		for _, group := range tree {
			if !slices.Contains(groups, group.Code) {
				continue
			}

			for _, child := range group.Childs {
				item := filter{
					group: group.Code,
					code:  child.Code,
					name:  strings.TrimSpace(child.Name),
				}
				if !seen[item.key()] {
					seen[item.key()] = true

					joined = append(joined, item)
				}
			}
		}
	}

	return joined, nil
}

// members returns the filters that list each offer id, in the order of the filters.
func (a *Adapter) members(
	ctx context.Context,
	locale string,
	filters []filter,
) (map[string][]filter, error) {
	lists := make([][]tbcoffers.Entry, len(filters))

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(parallelLists)

	for i, item := range filters {
		group.Go(func() error {
			entries, err := a.list(groupCtx, locale, item.key())
			lists[i] = entries

			return err
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("reading the offers of the filters of TBC: %w", err)
	}

	members := map[string][]filter{}

	for i, item := range filters {
		for _, entry := range lists[i] {
			members[entry.ID] = append(members[entry.ID], item)
		}
	}

	return members, nil
}

// list reads every page of the offers of one filter, or of every offer for none.
func (a *Adapter) list(ctx context.Context, locale, key string) ([]tbcoffers.Entry, error) {
	request := tbcoffers.OffersRequest{Locale: locale, PageSize: pageSize}
	if key != "" {
		request.Filter = []string{key}
	}

	var entries []tbcoffers.Entry

	for index := range maxPages {
		request.PageIndex = index

		page, err := a.client.Offers(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("reading the offers of %q: %w", key, err)
		}

		entries = append(entries, page.List...)

		if !page.PagingDetails.HasNextPage {
			break
		}
	}

	return entries, nil
}

func (a *Adapter) toOffer(
	ctx context.Context,
	entry tbcoffers.Entry,
	filters []filter,
	language kernel.Language,
) (domain.Offer, error) {
	start, err := dateOf(entry.StartDate)
	if err != nil {
		return domain.Offer{}, err
	}

	end, err := dateOf(entry.EndDate)
	if err != nil {
		return domain.Offer{}, err
	}

	period, err := domain.NewPeriod(start, end)
	if err != nil {
		return domain.Offer{}, fmt.Errorf("reading the period: %w", err)
	}

	tiers := a.tiersOf(ctx, entry.Segments)

	offer := domain.Offer{
		Bank:    a.Bank().Code,
		ID:      entry.ID,
		Title:   strings.TrimSpace(entry.Title),
		Benefit: strings.TrimSpace(entry.Description),
		Tiers:   tiers,
		Period:  period,
		URL:     addressOf(tiers, entry.Slug, language),
	}

	if entry.Partner != nil {
		offer.Partner = strings.TrimSpace(entry.Partner.Title)
	}

	a.applyFilters(ctx, &offer, filters)

	valid, err := domain.NewOffer(offer)
	if err != nil {
		return domain.Offer{}, fmt.Errorf("checking the offer: %w", err)
	}

	return valid, nil
}

// applyFilters gives the offer the values of each filter that lists it. An offer with
// no category or no kind gets other.
func (a *Adapter) applyFilters(ctx context.Context, offer *domain.Offer, filters []filter) {
	for _, item := range filters {
		switch item.group {
		case groupCategory:
			offer.Categories = append(offer.Categories, a.categoryOf(ctx, item.code))
			offer.BankCategories = append(offer.BankCategories, item.name)
		case groupOfferType:
			offer.Kinds = append(offer.Kinds, a.kindsOf(ctx, item.code)...)
		default:
			offer.Cards = append(offer.Cards, a.cardOf(ctx, item))
		}
	}

	if len(offer.Categories) == 0 {
		offer.Categories = []domain.Category{domain.CategoryOther}
	}

	if len(offer.Kinds) == 0 {
		offer.Kinds = []domain.BenefitKind{domain.KindOther}
	}
}

func (a *Adapter) tiersOf(ctx context.Context, segments []tbcoffers.Segment) []domain.Tier {
	var tiers []domain.Tier

	for _, segment := range segments {
		tier, ok := a.tiers[segment.ID]
		if !ok {
			a.warnOnce(ctx, "segment", segment.ID)

			continue
		}

		if !slices.Contains(tiers, tier) {
			tiers = append(tiers, tier)
		}
	}

	return tiers
}

func (a *Adapter) categoryOf(ctx context.Context, code string) domain.Category {
	category, ok := a.categories[code]
	if !ok {
		a.warnOnce(ctx, groupCategory, code)

		return domain.CategoryOther
	}

	return category
}

func (a *Adapter) kindsOf(ctx context.Context, code string) []domain.BenefitKind {
	kinds, ok := a.kinds[code]
	if !ok {
		a.warnOnce(ctx, groupOfferType, code)
	}

	return kinds
}

func (a *Adapter) cardOf(ctx context.Context, item filter) domain.Card {
	card, ok := a.cards[item.key()]
	if !ok {
		a.warnOnce(ctx, item.group, item.code)

		return domain.Card{Code: domain.CardCodeOf(item.code), Name: item.name}
	}

	return card
}

// warnOnce logs one warning for each code that no table holds.
func (a *Adapter) warnOnce(ctx context.Context, field, code string) {
	if _, seen := a.warned.LoadOrStore(field+"/"+code, struct{}{}); !seen {
		a.logger.WarnContext(ctx, "TBC sends a code that the server does not know",
			slog.String("field", field), slog.String("code", code))
	}
}

// addressOf returns the page of an offer: the site of Concept for an offer of Concept
// only, else the site of TBC Bank.
func addressOf(tiers []domain.Tier, slug string, language kernel.Language) string {
	if slices.Equal(tiers, []domain.Tier{domain.TierPremium}) {
		conceptLanguage := "ge"
		if language == kernel.LanguageEnglish {
			conceptLanguage = "en"
		}

		return fmt.Sprintf(conceptURL, conceptLanguage, url.PathEscape(slug))
	}

	return fmt.Sprintf(bankURL, language, url.PathEscape(slug))
}

// dateOf reads the day at the start of a date of TBC: "2026-10-20" of
// "2026-10-20T00:00:00". An empty text is the zero date.
func dateOf(text string) (kernel.Date, error) {
	if text == "" {
		return kernel.Date{}, nil
	}

	date, err := kernel.ParseDate(text[:min(len(text), dateLength)])
	if err != nil {
		return kernel.Date{}, fmt.Errorf("reading the date %q: %w", text, err)
	}

	return date, nil
}

func localeOf(language kernel.Language) string {
	if language == kernel.LanguageGeorgian {
		return "ka-GE"
	}

	return "en-US"
}

func unavailable(err error) error {
	return fmt.Errorf("reading the offers of TBC: %w: %w", kernel.ErrSourceUnavailable, err)
}
