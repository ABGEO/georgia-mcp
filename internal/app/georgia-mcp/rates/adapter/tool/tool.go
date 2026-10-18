package tool

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	source        = "National Bank of Georgia"
	maxCurrencies = 50

	getRatesDescription = "Returns the official exchange rates of the lari that the National Bank of Georgia " +
		"(NBG) sets, for one date. Use it for the rate of a currency today or on a past date, for a " +
		"contract, an invoice, or a tax form. Each rate is the price in GEL of `quantity` units; " +
		"`rate_per_unit` is the price of one unit. The NBG publishes the rates of each business day " +
		"at about 17:00 Tbilisi time, in force from the next day until the next set. A weekend or a " +
		"holiday returns the set in force on it. The official rate is indicative. It is not the " +
		"price of a bank."

	listHistoryDescription = "Returns the official exchange rates of one currency over a period of up to " +
		"31 days, as set by the National Bank of Georgia (NBG). Use it for the movement of a currency " +
		"over a week or a month. The answer holds one point for each set in force in the period, the " +
		"set in force on the first day included. Each rate is the price in GEL of `quantity` units."
)

// RatesGetter returns the official rates of a date.
type RatesGetter interface {
	Get(ctx context.Context, query app.OfficialRatesQuery) (app.OfficialRates, error)
}

// HistoryLister returns the official rates of one currency over a period.
type HistoryLister interface {
	List(ctx context.Context, query app.OfficialHistoryQuery) (app.OfficialHistory, error)
}

// Register adds the two tools of the official rates.
func Register(registry *mcpserver.Registry, rates RatesGetter, history HistoryLister) error {
	tools := handlers{rates: rates, history: history}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name:        "rates_get_official_rates",
		Title:       "Official exchange rates of the NBG",
		Description: getRatesDescription,
	}, tools.getRates)
	if err != nil {
		return err
	}

	return mcpserver.AddTool(registry, &mcp.Tool{
		Name:        "rates_list_official_history",
		Title:       "Official exchange rate history of the NBG",
		Description: listHistoryDescription,
	}, tools.listHistory)
}

type handlers struct {
	rates   RatesGetter
	history HistoryLister
}

func (h handlers) getRates(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getOfficialRatesInput,
) (*mcp.CallToolResult, officialRatesOutput, error) {
	query, err := toRatesQuery(in)
	if err != nil {
		return nil, officialRatesOutput{}, err
	}

	rates, err := h.rates.Get(ctx, query)
	if err != nil {
		return nil, officialRatesOutput{}, toolError(err)
	}

	return nil, toRatesOutput(rates), nil
}

func (h handlers) listHistory(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listOfficialHistoryInput,
) (*mcp.CallToolResult, officialHistoryOutput, error) {
	query, err := toHistoryQuery(in)
	if err != nil {
		return nil, officialHistoryOutput{}, err
	}

	history, err := h.history.List(ctx, query)
	if errors.Is(err, app.ErrPeriodReversed) {
		return nil, officialHistoryOutput{}, periodReversed(query.From, query.To, err)
	}

	if err != nil {
		return nil, officialHistoryOutput{}, toolError(err)
	}

	return nil, toHistoryOutput(history), nil
}

func toRatesQuery(in getOfficialRatesInput) (app.OfficialRatesQuery, error) {
	if len(in.Currencies) > maxCurrencies {
		return app.OfficialRatesQuery{}, tooManyCodes(len(in.Currencies))
	}

	codes := make([]domain.CurrencyCode, 0, len(in.Currencies))

	for _, text := range in.Currencies {
		code, err := domain.ParseCurrencyCode(text)
		if err != nil {
			return app.OfficialRatesQuery{}, invalidCode("currencies", text, err)
		}

		codes = append(codes, code)
	}

	language, err := parseLanguage(in.Language)
	if err != nil {
		return app.OfficialRatesQuery{}, err
	}

	query := app.OfficialRatesQuery{Currencies: codes, Language: language}

	if in.Date != "" {
		date, err := parseDate("date", in.Date)
		if err != nil {
			return app.OfficialRatesQuery{}, err
		}

		query.Date = &date
	}

	return query, nil
}

func toHistoryQuery(in listOfficialHistoryInput) (app.OfficialHistoryQuery, error) {
	code, err := domain.ParseCurrencyCode(in.Currency)
	if err != nil {
		return app.OfficialHistoryQuery{}, invalidCode("currency", in.Currency, err)
	}

	from, err := parseDate("from", in.From)
	if err != nil {
		return app.OfficialHistoryQuery{}, err
	}

	to, err := parseDate("to", in.To)
	if err != nil {
		return app.OfficialHistoryQuery{}, err
	}

	language, err := parseLanguage(in.Language)
	if err != nil {
		return app.OfficialHistoryQuery{}, err
	}

	return app.OfficialHistoryQuery{Currency: code, From: from, To: to, Language: language}, nil
}

func parseDate(field, text string) (domain.Date, error) {
	date, err := domain.ParseDate(text)
	if err != nil {
		return domain.Date{}, invalidDate(field, text, err)
	}

	return date, nil
}

func parseLanguage(text string) (kernel.Language, error) {
	language, err := kernel.ParseLanguage(text)
	if err != nil {
		return "", invalidLanguage(text, err)
	}

	return language, nil
}

func toRatesOutput(rates app.OfficialRates) officialRatesOutput {
	out := officialRatesOutput{
		Source: source,
		AsOf:   instant(rates.ReadAt),
		ReadAt: instant(rates.ReadAt),
		Stale:  rates.Stale,
		Rates:  make([]officialRateOutput, 0, len(rates.Set.Rates)),
	}

	if !rates.Set.Empty() {
		out.AsOf = instant(rates.Set.PublishedAt)
		out.ValidFrom = rates.Set.ValidFrom.String()
	}

	for _, rate := range rates.Set.Rates {
		out.Rates = append(out.Rates, toRateOutput(rate, rates.Set.ValidFrom))
	}

	for _, code := range rates.Missing {
		out.Missing = append(out.Missing, string(code))
	}

	slices.Sort(out.Missing)

	return out
}

func toHistoryOutput(history app.OfficialHistory) officialHistoryOutput {
	out := officialHistoryOutput{
		Source:   source,
		ReadAt:   instant(history.ReadAt),
		Stale:    history.Stale,
		Currency: string(history.Currency),
		Points:   make([]officialRateOutput, 0, len(history.Points)),
	}

	out.AsOf = out.ReadAt

	for _, point := range history.Points {
		out.Points = append(out.Points, toRateOutput(point.Rate, point.ValidFrom))
		out.Name = point.Rate.Name
		out.AsOf = instant(point.PublishedAt)
	}

	return out
}

func toRateOutput(rate domain.OfficialRate, validFrom domain.Date) officialRateOutput {
	return officialRateOutput{
		Currency:    string(rate.Currency),
		Name:        rate.Name,
		Quantity:    rate.Quantity,
		Rate:        exact(rate.Rate),
		RatePerUnit: exact(rate.RatePerUnit()),
		Change:      exact(rate.Change),
		ValidFrom:   validFrom.String(),
	}
}

// exact keeps the trailing zeros that String drops: the NBG publishes 2.6000.
func exact(value decimal.Decimal) string {
	return value.StringFixed(max(0, -value.Exponent()))
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(domain.TbilisiZone()).Format(time.RFC3339Nano)
}
