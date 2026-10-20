package tool

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	minLimit       = 1
	maxLimit       = 30
	defaultLimit   = 10
	maxQueryLength = 100
	maxRangeDays   = 366

	meaning = "benefit is the text of the bank. The server reads no number from it: read the " +
		"percent, the cap, and the conditions from the text. kinds come from a field of the bank, " +
		"and other means that the bank states none. tiers: standard is every customer, premium " +
		"the premium package (SOLO at BOG), private the private banking (Wealth Management at " +
		"BOG), youth the pupils and the students."

	searchDescription = "Finds the current offers that Georgian banks give to card holders at " +
		"partners: discounts, cashback, and points. Filters by bank, customer tier, category, " +
		"benefit kind, card, and period. For \"this month\", give the first and the last date of " +
		"the month. When a bank has a next_offset, call again with that offset to read more. " +
		"The benefit is the text of the bank. Compare the banks by reading it. Give " +
		"the person the url of each offer that you name. %s The banks of this server: %s. The " +
		"card codes of each bank: %s."

	getDescription = "Returns the full conditions of one offer of a Georgian bank. Use the bank " +
		"and the id from offers_search_offers. Give the person the url of the offer with the " +
		"conditions."
)

// OfferSearcher finds the current offers of the banks of the server.
type OfferSearcher interface {
	Search(ctx context.Context, query app.SearchQuery) (app.SearchResult, error)
	Banks() []domain.Bank
	Cards() map[domain.BankCode][]domain.Card
}

// OfferGetter returns the terms of one current offer.
type OfferGetter interface {
	Get(ctx context.Context, query app.OfferQuery) (app.OfferDetail, error)
}

// Register adds the tools of the card offers. The description of the search lists the
// banks and the card codes of the searcher.
func Register(registry *mcpserver.Registry, searcher OfferSearcher, getter OfferGetter) error {
	tools := handler{searcher: searcher, getter: getter}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name:  "offers_search_offers",
		Title: "Card offers of Georgian banks",
		Description: fmt.Sprintf(
			searchDescription, meaning, bankList(searcher.Banks()), cardList(searcher),
		),
	}, tools.search)
	if err != nil {
		return fmt.Errorf("adding the search of the offers: %w", err)
	}

	err = mcpserver.AddTool(registry, &mcp.Tool{
		Name:        "offers_get_offer",
		Title:       "Terms of a card offer",
		Description: getDescription,
	}, tools.get)
	if err != nil {
		return fmt.Errorf("adding the get of an offer: %w", err)
	}

	return nil
}

type handler struct {
	searcher OfferSearcher
	getter   OfferGetter
}

func (h handler) search(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchOffersInput,
) (*mcp.CallToolResult, searchOutput, error) {
	query, err := h.toSearchQuery(in)
	if err != nil {
		return nil, searchOutput{}, err
	}

	result, err := h.searcher.Search(ctx, query)
	if err != nil {
		return nil, searchOutput{}, failure(
			"banks",
			h.requested(query.Banks),
			h.searcher.Banks(),
			"",
			err,
		)
	}

	return nil, toSearchOutput(result), nil
}

func (h handler) get(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getOfferInput,
) (*mcp.CallToolResult, offerDetailOutput, error) {
	language, err := kernel.ParseLanguage(in.Language)
	if err != nil {
		return nil, offerDetailOutput{}, invalidLanguage(in.Language, err)
	}

	code, err := domain.ParseBankCode(in.Bank)
	if err != nil {
		return nil, offerDetailOutput{}, invalidBankCode("bank", in.Bank, h.searcher.Banks(), err)
	}

	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, offerDetailOutput{}, emptyID()
	}

	detail, err := h.getter.Get(ctx, app.OfferQuery{Bank: code, ID: id, Language: language})
	if err != nil {
		requested := h.requested([]domain.BankCode{code})

		return nil, offerDetailOutput{}, failure("bank", requested, h.searcher.Banks(), id, err)
	}

	return nil, offerDetailOutput{
		Source: detail.Bank.Name,
		AsOf:   instant(detail.ReadAt),
		Stale:  detail.Stale,
		Offer:  toOfferOutput(detail.Offer),
		Terms:  detail.Terms,
	}, nil
}

func (h handler) toSearchQuery(in searchOffersInput) (app.SearchQuery, error) {
	language, err := kernel.ParseLanguage(in.Language)
	if err != nil {
		return app.SearchQuery{}, invalidLanguage(in.Language, err)
	}

	query := app.SearchQuery{Language: language}

	if query.Limit, query.Offset, err = parsePage(in.Limit, in.Offset); err != nil {
		return app.SearchQuery{}, err
	}

	if in.Query != "" {
		query.Text = strings.TrimSpace(in.Query)
		if query.Text == "" || utf8.RuneCountInString(query.Text) > maxQueryLength {
			return app.SearchQuery{}, invalidQuery()
		}
	}

	if query.From, err = parseDate("from", in.From); err != nil {
		return app.SearchQuery{}, err
	}

	if query.To, err = parseDate("to", in.To); err != nil {
		return app.SearchQuery{}, err
	}

	return query, h.parseFilters(in, &query)
}

func (h handler) parseFilters(in searchOffersInput, query *app.SearchQuery) error {
	for _, text := range in.Banks {
		code, err := domain.ParseBankCode(text)
		if err != nil {
			return invalidBankCode("banks", text, h.searcher.Banks(), err)
		}

		query.Banks = append(query.Banks, code)
	}

	var err error

	if query.Tiers, err = parseAll(
		in.Tiers,
		domain.ParseTier,
		"tiers",
		"a tier",
		domain.Tiers(),
	); err != nil {
		return err
	}

	query.Categories, err = parseAll(
		in.Categories, domain.ParseCategory, "categories", "an offer category", domain.Categories(),
	)
	if err != nil {
		return err
	}

	query.Kinds, err = parseAll(
		in.Kinds, domain.ParseBenefitKind, "kinds", "a benefit kind", domain.BenefitKinds(),
	)
	if err != nil {
		return err
	}

	for _, text := range in.Cards {
		code, err := domain.ParseCardCode(text)
		if err != nil {
			return invalidCardCode(text, err)
		}

		query.Cards = append(query.Cards, code)
	}

	return nil
}

// parsePage returns the limit, 10 for zero, and the offset of a search.
func parsePage(limit, offset int) (int, int, error) {
	if offset < 0 {
		return 0, 0, invalidOffset(offset)
	}

	if limit == 0 {
		limit = defaultLimit
	}

	if limit < minLimit || limit > maxLimit {
		return 0, 0, invalidLimit(limit)
	}

	return limit, offset, nil
}

func parseAll[V ~string](
	texts []string,
	parse func(string) (V, error),
	field, noun string,
	accepted []V,
) ([]V, error) {
	values := make([]V, 0, len(texts))

	for _, text := range texts {
		value, err := parse(text)
		if err != nil {
			return nil, invalidValue(field, text, noun, joinValues(accepted), err)
		}

		values = append(values, value)
	}

	if len(values) == 0 {
		return nil, nil
	}

	return values, nil
}

func parseDate(field, text string) (kernel.Date, error) {
	if text == "" {
		return kernel.Date{}, nil
	}

	date, err := kernel.ParseDate(text)
	if err != nil {
		return kernel.Date{}, invalidDate(field, text, err)
	}

	return date, nil
}

// requested returns the banks of the codes, or every bank for none.
func (h handler) requested(codes []domain.BankCode) []domain.Bank {
	return slices.DeleteFunc(h.searcher.Banks(), func(bank domain.Bank) bool {
		return len(codes) > 0 && !slices.Contains(codes, bank.Code)
	})
}

func toSearchOutput(result app.SearchResult) searchOutput {
	out := searchOutput{Meaning: meaning, Banks: make([]bankOffersOutput, 0, len(result.Banks))}

	for _, bank := range result.Banks {
		bankOut := bankOffersOutput{
			Bank:       string(bank.Bank.Code),
			Source:     bank.Bank.Name,
			AsOf:       instant(bank.ReadAt),
			Stale:      bank.Stale,
			Total:      bank.Total,
			NextOffset: bank.NextOffset,
			Offers:     make([]offerOutput, 0, len(bank.Offers)),
		}

		for _, offer := range bank.Offers {
			bankOut.Offers = append(bankOut.Offers, toOfferOutput(offer))
		}

		out.Banks = append(out.Banks, bankOut)
	}

	for _, bank := range result.Unavailable {
		out.Unavailable = append(
			out.Unavailable,
			bankOutput{Bank: string(bank.Code), Source: bank.Name},
		)
	}

	return out
}

func toOfferOutput(offer domain.Offer) offerOutput {
	out := offerOutput{
		Bank:           string(offer.Bank),
		ID:             offer.ID,
		Title:          offer.Title,
		Partner:        offer.Partner,
		Benefit:        offer.Benefit,
		Kinds:          texts(offer.Kinds),
		Categories:     texts(offer.Categories),
		BankCategories: offer.BankCategories,
		Tiers:          texts(offer.Tiers),
		URL:            offer.URL,
	}

	for _, card := range offer.Cards {
		out.Cards = append(out.Cards, cardOutput{Code: string(card.Code), Name: card.Name})
	}

	if !offer.Period.Start.IsZero() {
		out.StartDate = offer.Period.Start.String()
	}

	if !offer.Period.End.IsZero() {
		out.EndDate = offer.Period.End.String()
	}

	return out
}

func texts[V ~string](values []V) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

func joinValues[V ~string](values []V) string {
	return strings.Join(texts(values), ", ")
}

// bankList lists each bank with its name: "bog (Bank of Georgia)".
func bankList(banks []domain.Bank) string {
	entries := make([]string, 0, len(banks))
	for _, bank := range banks {
		entries = append(entries, fmt.Sprintf("%s (%s)", bank.Code, bank.Name))
	}

	return strings.Join(entries, ", ")
}

// cardList lists the cards of each bank: "bog: amex (Amex), visa (Visa)".
func cardList(searcher OfferSearcher) string {
	cards := searcher.Cards()
	entries := make([]string, 0, len(cards))

	for _, bank := range searcher.Banks() {
		named := make([]string, 0, len(cards[bank.Code]))
		for _, card := range cards[bank.Code] {
			named = append(named, fmt.Sprintf("%s (%s)", card.Code, card.Name))
		}

		entries = append(entries, string(bank.Code)+": "+strings.Join(named, ", "))
	}

	return strings.Join(entries, "; ")
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339Nano)
}
