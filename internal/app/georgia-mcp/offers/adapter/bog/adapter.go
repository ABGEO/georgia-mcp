// Package bog reads the card offers from the client of Bank of Georgia and converts its
// answer to the domain. It is the anti-corruption layer of the context for BOG.
package bog

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/text"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/pkg/bogoffers"
)

const (
	pageSize = 100
	// maxPages bounds a read. BOG listed 239 offers in 3 pages on 2026-10-20.
	maxPages   = 20
	codeSep    = ";"
	detailsURL = "https://bankofgeorgia.ge/%s/offers-hub/details/%d"
)

var errFailed = errors.New("bog: the answer reports a failure")

// Adapter implements app.OfferSource with the client of Bank of Georgia.
type Adapter struct {
	client     *bogoffers.Client
	now        func() time.Time
	logger     *slog.Logger
	warned     sync.Map
	tiers      map[string]domain.Tier
	kinds      map[string]domain.BenefitKind
	categories map[int]domain.Category
	cards      map[string]domain.Card
}

var _ app.OfferSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each code that no table of the adapter holds,
// and one for each offer that it leaves out.
func NewAdapter(client *bogoffers.Client, now func() time.Time, logger *slog.Logger) *Adapter {
	return &Adapter{
		client:     client,
		now:        now,
		logger:     logger,
		tiers:      tierTable(),
		kinds:      kindTable(),
		categories: categoryTable(),
		cards:      cardTable(),
	}
}

// Bank returns Bank of Georgia.
func (a *Adapter) Bank() domain.Bank {
	return domain.Bank{Code: "bog", Name: "Bank of Georgia"}
}

// Cards returns the cards of the table of the adapter, sorted by code.
func (a *Adapter) Cards() []domain.Card {
	return slices.SortedFunc(maps.Values(a.cards), func(left, right domain.Card) int {
		return cmp.Compare(left.Code, right.Code)
	})
}

// Offers reads every page in one language, and leaves out each offer that the domain
// refuses. Every failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) Offers(ctx context.Context, language kernel.Language) (app.Catalog, error) {
	var offers []bogoffers.Offer

	for number := range maxPages {
		page, err := a.client.Offers(ctx, bogoffers.OffersQuery{
			Language:   string(language),
			PageNumber: number,
			PageSize:   pageSize,
		})
		if err == nil && page.Code != 0 {
			err = fmt.Errorf("%w: page %d, code %d", errFailed, number, page.Code)
		}

		if err != nil {
			return app.Catalog{}, fmt.Errorf(
				"reading the offers of BOG: %w: %w", kernel.ErrSourceUnavailable, err,
			)
		}

		offers = append(offers, page.Result.Offers...)

		if number+1 >= page.Result.TotalPageCount || len(page.Result.Offers) == 0 {
			break
		}
	}

	catalog := app.Catalog{Offers: make([]domain.Offer, 0, len(offers)), ReadAt: a.now()}

	for _, vendor := range offers {
		offer, err := a.toOffer(ctx, vendor, language)
		if err != nil {
			a.logger.WarnContext(ctx, "BOG lists an offer that the server leaves out",
				slog.Int64("id", vendor.CampaignID), slog.String("cause", err.Error()))

			continue
		}

		catalog.Offers = append(catalog.Offers, offer)
	}

	return catalog, nil
}

// Terms returns the terms of the offer. The list of BOG holds them, so the use case does
// not call this method for an offer of BOG.
func (a *Adapter) Terms(
	_ context.Context,
	offer domain.Offer,
	_ kernel.Language,
) (app.Terms, error) {
	return app.Terms{Text: offer.Terms, ReadAt: a.now()}, nil
}

func (a *Adapter) toOffer(
	ctx context.Context,
	vendor bogoffers.Offer,
	language kernel.Language,
) (domain.Offer, error) {
	period, err := domain.NewPeriod(dateOf(vendor.StartDate), dateOf(vendor.EndDate))
	if err != nil {
		return domain.Offer{}, fmt.Errorf("reading the period: %w", err)
	}

	offer := domain.Offer{
		Bank:       a.Bank().Code,
		ID:         strconv.FormatInt(vendor.CampaignID, 10),
		Title:      strings.TrimSpace(vendor.Title),
		Benefit:    strings.TrimSpace(vendor.ShortDesc),
		Kinds:      []domain.BenefitKind{a.kindOf(ctx, vendor.BenefitName)},
		Categories: []domain.Category{a.categoryOf(ctx, vendor.CategoryID)},
		Tiers:      a.tiersOf(ctx, vendor.SegmentTypes),
		Cards:      a.cardsOf(ctx, vendor.ProductCodes),
		Period:     period,
		URL:        fmt.Sprintf(detailsURL, language, vendor.CampaignID),
		Terms:      text.FromHTML(vendor.LongDesc),
	}

	if name := strings.TrimSpace(vendor.CategoryDesc); name != "" {
		offer.BankCategories = []string{name}
	}

	valid, err := domain.NewOffer(offer)
	if err != nil {
		return domain.Offer{}, fmt.Errorf("checking the offer: %w", err)
	}

	return valid, nil
}

// dateOf returns the date in Tbilisi of milliseconds of the Unix epoch, or the zero date.
func dateOf(milliseconds *int64) kernel.Date {
	if milliseconds == nil {
		return kernel.Date{}
	}

	return kernel.Today(time.UnixMilli(*milliseconds))
}

func (a *Adapter) tiersOf(ctx context.Context, segments string) []domain.Tier {
	var tiers []domain.Tier

	for _, segment := range splitCodes(segments) {
		tier, ok := a.tiers[segment]
		if !ok {
			a.warnOnce(ctx, "segment", segment)

			continue
		}

		tiers = append(tiers, tier)
	}

	return tiers
}

func (a *Adapter) cardsOf(ctx context.Context, products string) []domain.Card {
	codes := splitCodes(products)
	cards := make([]domain.Card, 0, len(codes))

	for _, product := range codes {
		card, ok := a.cards[product]
		if !ok {
			a.warnOnce(ctx, "product", product)

			card = domain.Card{Code: domain.CardCodeOf(product), Name: product}
		}

		cards = append(cards, card)
	}

	return cards
}

func (a *Adapter) kindOf(ctx context.Context, benefit string) domain.BenefitKind {
	kind, ok := a.kinds[benefit]
	if !ok {
		a.warnOnce(ctx, "benefit", benefit)

		return domain.KindOther
	}

	return kind
}

func (a *Adapter) categoryOf(ctx context.Context, id int) domain.Category {
	category, ok := a.categories[id]
	if !ok {
		a.warnOnce(ctx, "category", strconv.Itoa(id))

		return domain.CategoryOther
	}

	return category
}

// warnOnce logs one warning for each code that no table holds.
func (a *Adapter) warnOnce(ctx context.Context, field, code string) {
	if _, seen := a.warned.LoadOrStore(field+"/"+code, struct{}{}); !seen {
		a.logger.WarnContext(ctx, "BOG sends a code that the server does not know",
			slog.String("field", field), slog.String("code", code))
	}
}

func splitCodes(codes string) []string {
	var out []string

	for code := range strings.SplitSeq(codes, codeSep) {
		if code = strings.TrimSpace(code); code != "" {
			out = append(out, code)
		}
	}

	return out
}
