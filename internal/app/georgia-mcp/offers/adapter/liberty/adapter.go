// Package liberty reads the card offers of Studio from the client of Liberty and converts
// its answer to the domain. It is the anti-corruption layer of the context for Liberty.
package liberty

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sync/errgroup"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/pkg/liberty"
)

const (
	pageSize = 16
	// maxPages bounds a list. Liberty listed 206 offers in 13 pages on 2026-10-20.
	maxPages = 30
	// parallelLists bounds the lists of a category or a type that run at once.
	parallelLists = 4
	labelSep      = "; "
	detailsURL    = "https://studioliberty.ge/%s/studio-lifestyle/offers/details/%d"
)

var daysLeft = regexp.MustCompile(`\d+`)

// list is one list of Liberty: the offers of one category or of one type.
type list struct {
	categoryID, typeID int
	title              string
}

// Adapter implements app.OfferSource with the client of Liberty.
type Adapter struct {
	client     *liberty.Client
	now        func() time.Time
	logger     *slog.Logger
	warned     sync.Map
	categories map[int]domain.Category
	kinds      map[int]domain.BenefitKind
	cards      map[int]domain.Card
}

var _ app.OfferSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each id that no table of the adapter holds, and
// one for each offer that it leaves out.
func NewAdapter(client *liberty.Client, now func() time.Time, logger *slog.Logger) *Adapter {
	return &Adapter{
		client:     client,
		now:        now,
		logger:     logger,
		categories: categoryTable(),
		kinds:      kindTable(),
		cards:      cardTable(),
	}
}

// Bank returns Liberty Bank.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "liberty", Name: "Liberty Bank"}
}

// Cards returns the cards of the table of the adapter, sorted by code.
func (a *Adapter) Cards() []domain.Card {
	return slices.SortedFunc(maps.Values(a.cards), func(left, right domain.Card) int {
		return cmp.Compare(left.Code, right.Code)
	})
}

// Offers reads the categories, the types, every offer, and the offers of each category
// and each type, and gives each offer the values of the lists that hold it. Every
// failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) Offers(ctx context.Context, language kernel.Language) (app.Catalog, error) {
	readAt := a.now()

	lists, err := a.lists(ctx, language)
	if err != nil {
		return app.Catalog{}, unavailable(err)
	}

	offers, err := a.pages(ctx, liberty.OffersQuery{Language: string(language)})
	if err != nil {
		return app.Catalog{}, unavailable(err)
	}

	members, err := a.members(ctx, language, lists)
	if err != nil {
		return app.Catalog{}, unavailable(err)
	}

	catalog := app.Catalog{Offers: make([]domain.Offer, 0, len(offers)), ReadAt: readAt}
	today := kernel.Today(readAt)

	for _, vendor := range offers {
		offer, err := a.toOffer(ctx, vendor, members[vendor.ID], language, today)
		if err != nil {
			a.logger.WarnContext(ctx, "Liberty lists an offer that the server leaves out",
				slog.Int("id", vendor.ID), slog.String("cause", err.Error()))

			continue
		}

		catalog.Offers = append(catalog.Offers, offer)
	}

	return catalog, nil
}

// Terms reads the page of the offer and joins its paragraphs, one on each line.
func (a *Adapter) Terms(
	ctx context.Context,
	offer domain.Offer,
	language kernel.Language,
) (app.Terms, error) {
	id, err := strconv.Atoi(offer.ID)
	if err != nil {
		return app.Terms{}, unavailable(fmt.Errorf("reading the id %q: %w", offer.ID, err))
	}

	page, err := a.client.OfferPage(ctx, id, string(language))
	if err != nil {
		return app.Terms{}, unavailable(err)
	}

	lines := make([]string, 0, len(page.Paragraphs))

	for _, paragraph := range page.Paragraphs {
		if line := strings.TrimSpace(paragraph); line != "" {
			lines = append(lines, line)
		}
	}

	return app.Terms{Text: strings.Join(lines, "\n"), ReadAt: a.now()}, nil
}

// lists returns one list for each category and each type.
func (a *Adapter) lists(ctx context.Context, language kernel.Language) ([]list, error) {
	categories, err := a.client.Categories(ctx, string(language))
	if err != nil {
		return nil, fmt.Errorf("reading the categories of Liberty: %w", err)
	}

	types, err := a.client.Types(ctx, string(language))
	if err != nil {
		return nil, fmt.Errorf("reading the types of Liberty: %w", err)
	}

	lists := make([]list, 0, len(categories)+len(types))

	for _, category := range categories {
		lists = append(
			lists,
			list{categoryID: category.ID, title: strings.TrimSpace(category.Title)},
		)
	}

	for _, offerType := range types {
		lists = append(lists, list{typeID: offerType.ID, title: strings.TrimSpace(offerType.Title)})
	}

	return lists, nil
}

// members returns the lists that hold each offer id, in the order of the lists.
func (a *Adapter) members(
	ctx context.Context,
	language kernel.Language,
	lists []list,
) (map[int][]list, error) {
	offers := make([][]liberty.Offer, len(lists))

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(parallelLists)

	for i, item := range lists {
		group.Go(func() error {
			read, err := a.pages(groupCtx, liberty.OffersQuery{
				CategoryID: item.categoryID,
				TypeID:     item.typeID,
				Language:   string(language),
			})
			offers[i] = read

			return err
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("reading the offers of the lists of Liberty: %w", err)
	}

	members := map[int][]list{}

	for i, item := range lists {
		for _, offer := range offers[i] {
			members[offer.ID] = append(members[offer.ID], item)
		}
	}

	return members, nil
}

// pages reads every page of one query. A page with less than 16 offers is the last.
func (a *Adapter) pages(ctx context.Context, query liberty.OffersQuery) ([]liberty.Offer, error) {
	var offers []liberty.Offer

	for page := 1; page <= maxPages; page++ {
		query.Page = page

		read, err := a.client.Offers(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("reading the offers of Liberty: %w", err)
		}

		offers = append(offers, read...)

		if len(read) < pageSize {
			break
		}
	}

	return offers, nil
}

func (a *Adapter) toOffer(
	ctx context.Context,
	vendor liberty.Offer,
	lists []list,
	language kernel.Language,
	today kernel.Date,
) (domain.Offer, error) {
	offer := domain.Offer{
		Bank:    a.Bank().Code,
		ID:      strconv.Itoa(vendor.ID),
		Title:   titleOf(vendor),
		Benefit: benefitOf(vendor),
		Tiers:   []domain.Tier{domain.TierPremium},
		Period:  domain.Period{End: endOf(vendor.RemainingDays, today)},
		URL:     fmt.Sprintf(detailsURL, language, vendor.ID),
	}

	for _, item := range lists {
		a.apply(ctx, &offer, item)
	}

	if len(offer.Categories) == 0 {
		offer.Categories = []domain.Category{domain.CategoryOther}
	}

	if len(offer.Kinds) == 0 {
		offer.Kinds = []domain.BenefitKind{domain.KindOther}
	}

	valid, err := domain.NewOffer(offer)
	if err != nil {
		return domain.Offer{}, fmt.Errorf("checking the offer: %w", err)
	}

	return valid, nil
}

// apply gives the offer the value of one list that holds it.
func (a *Adapter) apply(ctx context.Context, offer *domain.Offer, item list) {
	if item.categoryID != 0 {
		category, ok := a.categories[item.categoryID]
		if !ok {
			a.warnOnce(ctx, "category", item.categoryID)

			category = domain.CategoryOther
		}

		offer.Categories = append(offer.Categories, category)
		offer.BankCategories = append(offer.BankCategories, item.title)

		return
	}

	if kind, ok := a.kinds[item.typeID]; ok {
		offer.Kinds = append(offer.Kinds, kind)

		return
	}

	card, ok := a.cards[item.typeID]
	if !ok {
		a.warnOnce(ctx, "type", item.typeID)

		card = domain.Card{
			Code: domain.CardCode(fmt.Sprintf("type_%d", item.typeID)),
			Name: item.title,
		}
	}

	offer.Cards = append(offer.Cards, card)
}

// warnOnce logs one warning for each id that no table holds.
func (a *Adapter) warnOnce(ctx context.Context, field string, id int) {
	if _, seen := a.warned.LoadOrStore(fmt.Sprintf("%s/%d", field, id), struct{}{}); !seen {
		a.logger.WarnContext(ctx, "Liberty sends an id that the server does not know",
			slog.String("field", field), slog.Int("id", id))
	}
}

// titleOf returns the title of an offer, or the words of the slug of its address when the
// title is empty: ".../details/724-the-westin-grande-sukhumvit" is "The Westin Grande
// Sukhumvit". Liberty leaves a title empty in one language.
func titleOf(vendor liberty.Offer) string {
	if title := strings.TrimSpace(vendor.Title); title != "" {
		return title
	}

	slug := path.Base(strings.TrimSpace(vendor.URL))

	_, words, found := strings.Cut(slug, "-")
	if !found {
		return ""
	}

	parts := strings.FieldsFunc(words, func(char rune) bool { return char == '-' })
	for i, word := range parts {
		first, size := utf8.DecodeRuneInString(word)
		parts[i] = string(unicode.ToUpper(first)) + word[size:]
	}

	return strings.Join(parts, " ")
}

// benefitOf joins the labels of an offer, or gives its short content when it has none.
func benefitOf(vendor liberty.Offer) string {
	labels := make([]string, 0, len(vendor.Labels))

	for _, label := range vendor.Labels {
		if label = strings.TrimSpace(label); label != "" {
			labels = append(labels, label)
		}
	}

	if len(labels) == 0 {
		return strings.TrimSpace(vendor.ShortContent)
	}

	return strings.Join(labels, labelSep)
}

// endOf returns the date that is the days left after today: "72 Days left" on 2026-10-20
// is 2026-12-31. A text with no number gives the zero date.
func endOf(remaining string, today kernel.Date) kernel.Date {
	days, err := strconv.Atoi(daysLeft.FindString(remaining))
	if err != nil {
		return kernel.Date{}
	}

	return today.AddDays(days)
}

func unavailable(err error) error {
	return fmt.Errorf("reading the offers of Liberty: %w: %w", kernel.ErrSourceUnavailable, err)
}
