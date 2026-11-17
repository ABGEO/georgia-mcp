package tool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	gnercSource = "Georgian National Energy and Water Supply Regulatory Commission (GNERC)"
	balanceUnit = "million kWh"
	priceUnit   = "tetri per kWh"

	getBalanceDescription = "Returns the monthly electricity balance of Georgia from the Georgian " +
		"National Energy and Water Supply Regulatory Commission (GNERC): the generation by type " +
		"of power plant, the import and the export by country, the supply, the consumption by " +
		"group, the losses, the domestic consumption, and the transit, in million kWh, for one " +
		"month from 2022-01, or the total of one year from 2022. GNERC publishes a month about " +
		"three weeks after it ends. With no input, it returns the newest published month. A year " +
		"that is not over holds only its published months: see months. Data from GNERC."

	listPricesDescription = "Returns three monthly electricity prices of Georgia from the Georgian " +
		"National Energy and Water Supply Regulatory Commission (GNERC), in tetri per kWh: the " +
		"weighted average price of balancing electricity, the guaranteed capacity fee, and the " +
		"price of the deregulated power plants. Give a period of at most 60 months from 2015-01. " +
		"With no input, it returns the last 12 published months. Data from GNERC."
)

func (h handler) getBalance(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getBalanceInput,
) (*mcp.CallToolResult, balanceOutput, error) {
	month, err := parseMonth("month", in.Month)
	if err != nil {
		return nil, balanceOutput{}, err
	}

	answer, err := h.uses.GetBalance.Get(ctx, app.BalanceQuery{Month: month, Year: in.Year})
	if err != nil {
		return nil, balanceOutput{}, gnercError(err, in.Month, "")
	}

	return nil, toBalanceOutput(answer), nil
}

func (h handler) listPrices(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in listPricesInput,
) (*mcp.CallToolResult, pricesOutput, error) {
	from, err := parseMonth("from", in.From)
	if err != nil {
		return nil, pricesOutput{}, err
	}

	to, err := parseMonth("to", in.To)
	if err != nil {
		return nil, pricesOutput{}, err
	}

	answer, err := h.uses.ListPrices.List(ctx, app.PriceQuery{From: from, To: to})
	if err != nil {
		return nil, pricesOutput{}, gnercError(err, in.From, in.To)
	}

	return nil, toPricesOutput(answer), nil
}

func parseMonth(field, text string) (domain.Month, error) {
	if text == "" {
		return domain.Month{}, nil
	}

	month, err := domain.ParseMonth(text)
	if err != nil {
		return domain.Month{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"%s: %q is not a month. Use YYYY-MM, for example 2026-03.",
				field,
				text,
			),
			Cause: err,
		}
	}

	return month, nil
}

// gnercError gives the model a message that names the cause and the next action. from
// and to are the ends of the period of the request.
func gnercError(err error, from, to string) error {
	var (
		periodErr *app.PeriodError
		spanErr   *app.SpanError
		message   string
	)

	switch {
	case errors.Is(err, app.ErrMonthAndYear):
		message = "month and year: give one of them, or neither for the newest published month."
	case errors.As(err, &periodErr):
		message = periodMessage(periodErr)
	case errors.Is(err, app.ErrReversedPeriod):
		message = fmt.Sprintf(
			"to: %s is before from, %s. Give a period that ends on or after its start.",
			to,
			from,
		)
	case errors.As(err, &spanErr):
		message = fmt.Sprintf("from and to: the period holds %d months. Give at most %d.",
			spanErr.Months, domain.MaxPriceMonths)
	case errors.Is(err, kernel.ErrSourceUnavailable):
		message = "GNERC did not answer. Try again in a few minutes."
	default:
		message = "The server failed to read the data of GNERC."
	}

	return &mcpserver.ToolError{Message: message, Cause: err}
}

func periodMessage(err *app.PeriodError) string {
	switch err.Field {
	case "year":
		return fmt.Sprintf("year: %s is outside the balance of GNERC. Give a year from %d to %d.",
			err.Value, err.First.Year, err.Newest.Year)
	case "month":
		return fmt.Sprintf("month: %s is outside the balance of GNERC. Give a month from %s to %s.",
			err.Value, err.First, err.Newest)
	default:
		return fmt.Sprintf("%s: %s is outside the prices of GNERC. Give a month from %s to %s.",
			err.Field, err.Value, err.First, err.Newest)
	}
}

// asOf is the last change of GNERC, or the read when GNERC does not say.
func asOf(modified, readAt time.Time) string {
	if modified.IsZero() {
		return instant(readAt)
	}

	return instant(modified)
}

func fixed(number decimal.Decimal) string {
	return number.StringFixed(domain.Precision)
}

func toPricesOutput(answer app.PriceAnswer) pricesOutput {
	out := pricesOutput{
		Source:      gnercSource,
		AsOf:        asOf(answer.Modified, answer.ReadAt),
		Stale:       answer.Stale,
		NewestMonth: answer.Newest.String(),
		Unit:        priceUnit,
		Months:      make([]monthPricesOutput, 0, len(answer.Months)),
	}

	price := func(number *decimal.Decimal) string {
		if number == nil {
			return ""
		}

		return fixed(*number)
	}

	for _, month := range answer.Months {
		out.Months = append(out.Months, monthPricesOutput{
			Month:       month.Month.String(),
			Balancing:   price(month.Balancing),
			CapacityFee: price(month.CapacityFee),
			Deregulated: price(month.Deregulated),
		})
	}

	return out
}

func toBalanceOutput(answer app.BalanceAnswer) balanceOutput {
	figures := answer.Figures
	value := func(item domain.Item) string {
		number, found := figures[item]
		if !found {
			return ""
		}

		return fixed(number)
	}

	out := balanceOutput{
		Source:      gnercSource,
		AsOf:        asOf(answer.Modified, answer.ReadAt),
		Stale:       answer.Stale,
		Period:      answer.Period,
		Months:      answer.Months,
		NewestMonth: answer.Newest.String(),
		Unit:        balanceUnit,
		Generation: generationBalance{
			Total: value(domain.GenerationTotal), Thermal: value(domain.GenerationThermal),
			Wind: value(domain.GenerationWind), Solar: value(domain.GenerationSolar),
			Hydro: value(domain.GenerationHydro),
		},
		StationOwnUse: value(domain.StationOwnUse),
		Import: countryBalance{
			Total: value(domain.ImportTotal), Russia: value(domain.ImportRussia),
			Turkey: value(domain.ImportTurkey), Azerbaijan: value(domain.ImportAzerbaijan),
			Armenia: value(domain.ImportArmenia), IntoAbkhazia: value(domain.ImportIntoAbkhazia),
		},
		Supply: value(domain.Supply),
		Consumption: consumptionBalance{
			Total: value(domain.ConsumptionTotal), Abkhazia: value(domain.ConsumptionAbkhazia),
			Direct:          value(domain.ConsumptionDirect),
			StoppedStations: value(domain.ConsumptionStoppedStations),
		},
		Export: countryBalance{
			Total: value(domain.ExportTotal), Russia: value(domain.ExportRussia),
			Turkey: value(domain.ExportTurkey), Azerbaijan: value(domain.ExportAzerbaijan),
			Armenia: value(domain.ExportArmenia),
		},
		DomesticConsumption: value(domain.DomesticConsumption),
	}

	out.Consumption.Suppliers, out.Losses, out.Transit = groupsOf(value)

	return out
}

// groupsOf returns each group of items that holds a value, and nil for an empty group.
func groupsOf(value func(domain.Item) string) (*suppliersBalance, *lossesBalance, *transitBalance) {
	var (
		suppliers *suppliersBalance
		losses    *lossesBalance
		transit   *transitBalance
	)

	if group := (suppliersBalance{
		Total: value(domain.ConsumptionSuppliers), UniversalService: value(domain.ConsumptionUniversalService),
		PublicService: value(domain.ConsumptionPublicService), FreeMarket: value(domain.ConsumptionFreeMarket),
	}); group != (suppliersBalance{}) {
		suppliers = &group
	}

	if group := (lossesBalance{
		Total: value(domain.LossesTotal), Transmission: value(domain.LossesTransmission),
		Distribution: value(domain.LossesDistribution),
	}); group != (lossesBalance{}) {
		losses = &group
	}

	if group := (transitBalance{
		Total: value(domain.TransitTotal), AzerbaijanToTurkey: value(domain.TransitAzerbaijanTurkey),
		RussiaToTurkey: value(domain.TransitRussiaTurkey), ArmeniaToTurkey: value(domain.TransitArmeniaTurkey),
	}); group != (transitBalance{}) {
		transit = &group
	}

	return suppliers, losses, transit
}
