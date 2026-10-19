package tool

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	meaning = "price: the price in GEL of one unit at a station with an attendant. " +
		"self_service_price: the price in GEL of one unit at a self-service station of the " +
		"same company. A product with no self_service_price has none."

	getPricesDescription = "Returns the current fuel prices of Georgian fuel companies: the price in GEL " +
		"of one unit of each product. Use it for the price of petrol, diesel, or gas at a company " +
		"today, or to compare companies. Each product has a grade that every company shares: %s. " +
		"%s The fuel companies of this server: %s. The prices are of today. The tool has no history."
)

// PricesGetter returns the fuel prices of the companies of the server.
type PricesGetter interface {
	Get(ctx context.Context, query app.PricesQuery) (app.Prices, error)
	Companies() []domain.Company
}

// Register adds the tool of the fuel prices. Its description lists the companies of
// the getter.
func Register(registry *mcpserver.Registry, prices PricesGetter) error {
	tool := handler{prices: prices}

	return mcpserver.AddTool(registry, &mcp.Tool{
		Name:  "fuel_get_prices",
		Title: "Fuel prices of Georgian fuel companies",
		Description: fmt.Sprintf(
			getPricesDescription, gradeUnits(), meaning, companyList(prices.Companies()),
		),
	}, tool.getPrices)
}

type handler struct {
	prices PricesGetter
}

func (h handler) getPrices(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getPricesInput,
) (*mcp.CallToolResult, pricesOutput, error) {
	language, err := kernel.ParseLanguage(in.Language)
	if err != nil {
		return nil, pricesOutput{}, invalidLanguage(in.Language, err)
	}

	query, err := h.toQuery(in)
	if err != nil {
		return nil, pricesOutput{}, err
	}

	prices, err := h.prices.Get(ctx, query)
	if err != nil {
		return nil, pricesOutput{}, h.toolError(query, err)
	}

	return nil, toOutput(prices, language), nil
}

func (h handler) toQuery(in getPricesInput) (app.PricesQuery, error) {
	var query app.PricesQuery

	for _, text := range in.Grades {
		grade, err := domain.ParseGrade(text)
		if err != nil {
			return app.PricesQuery{}, invalidGrade(text, err)
		}

		query.Grades = append(query.Grades, grade)
	}

	for _, text := range in.Companies {
		code, err := domain.ParseCompanyCode(text)
		if err != nil {
			return app.PricesQuery{}, invalidCompanyCode(text, h.prices.Companies(), err)
		}

		query.Companies = append(query.Companies, code)
	}

	return query, nil
}

func toOutput(prices app.Prices, language kernel.Language) pricesOutput {
	out := pricesOutput{
		Meaning:   meaning,
		Companies: make([]companyPricesOutput, 0, len(prices.Companies)),
	}

	for _, company := range prices.Companies {
		companyOut := companyPricesOutput{
			Company: string(company.Company.Code),
			Source:  company.Company.Name,
			AsOf:    instant(company.ReadAt),
			Stale:   company.Stale,
			Prices:  make([]priceOutput, 0, len(company.Prices)),
		}

		for _, price := range company.Prices {
			companyOut.Prices = append(companyOut.Prices, toPriceOutput(price, language))
		}

		for _, grade := range company.Missing {
			companyOut.Missing = append(companyOut.Missing, string(grade))
		}

		out.Companies = append(out.Companies, companyOut)
	}

	for _, company := range prices.Unavailable {
		out.Unavailable = append(
			out.Unavailable,
			companyOutput{Company: string(company.Code), Source: company.Name},
		)
	}

	return out
}

func toPriceOutput(price domain.ProductPrice, language kernel.Language) priceOutput {
	out := priceOutput{
		Product:       price.Name.In(language),
		Grade:         string(price.Grade),
		Unit:          string(price.Grade.Unit()),
		Price:         exact(price.Standard),
		EffectiveDate: price.EffectiveDate.String(),
	}

	if price.SelfService.Valid {
		out.SelfServicePrice = exact(price.SelfService.Decimal)
	}

	return out
}

// gradeUnits lists each grade with the unit of its price: "regular (litre), ...".
func gradeUnits() string {
	entries := make([]string, 0, len(domain.Grades()))
	for _, grade := range domain.Grades() {
		entries = append(entries, fmt.Sprintf("%s (%s)", grade, grade.Unit()))
	}

	return strings.Join(entries, ", ")
}

func companyList(companies []domain.Company) string {
	entries := make([]string, 0, len(companies))
	for _, company := range companies {
		entries = append(entries, fmt.Sprintf("%s (%s)", company.Code, company.Name))
	}

	return strings.Join(entries, ", ")
}

// exact keeps the digits that the company publishes.
func exact(value decimal.Decimal) string {
	return value.StringFixed(max(0, -value.Exponent()))
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339Nano)
}
