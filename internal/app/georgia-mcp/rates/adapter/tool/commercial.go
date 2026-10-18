package tool

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	meaning = "buy: the price in GEL that the bank pays when you sell it one unit. " +
		"sell: the price in GEL that the bank asks when you buy one unit from it."

	getCommercialDescription = "Returns the current commercial exchange rates of Georgian banks: " +
		"the prices in GEL at which a bank buys and sells one unit of a currency. Use it for the " +
		"price of a currency at a bank today, or to compare banks. Ask for 1 to 10 currencies. %s " +
		"The banks of this server: %s. The rates are of today. The tool has no history."
)

// CommercialRatesGetter returns the commercial rates of the banks of the server.
type CommercialRatesGetter interface {
	Get(ctx context.Context, query app.CommercialRatesQuery) (app.CommercialRates, error)
	Banks() []domain.Bank
}

// RegisterCommercial adds the tool of the commercial rates. Its description lists the
// banks of the getter.
func RegisterCommercial(registry *mcpserver.Registry, rates CommercialRatesGetter) error {
	tool := commercialHandler{rates: rates}

	return mcpserver.AddTool(registry, &mcp.Tool{
		Name:        "rates_get_commercial_rates",
		Title:       "Commercial exchange rates of Georgian banks",
		Description: fmt.Sprintf(getCommercialDescription, meaning, bankList(rates.Banks())),
	}, tool.getCommercialRates)
}

type commercialHandler struct {
	rates CommercialRatesGetter
}

func (h commercialHandler) getCommercialRates(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getCommercialRatesInput,
) (*mcp.CallToolResult, commercialRatesOutput, error) {
	query, err := h.toCommercialQuery(in)
	if err != nil {
		return nil, commercialRatesOutput{}, err
	}

	rates, err := h.rates.Get(ctx, query)
	if err != nil {
		return nil, commercialRatesOutput{}, h.commercialError(query, err)
	}

	return nil, toCommercialOutput(rates), nil
}

func (h commercialHandler) toCommercialQuery(
	in getCommercialRatesInput,
) (app.CommercialRatesQuery, error) {
	var query app.CommercialRatesQuery

	for _, text := range in.Currencies {
		code, err := domain.ParseCurrencyCode(text)
		if err != nil {
			return app.CommercialRatesQuery{}, invalidCode("currencies", text, err)
		}

		query.Currencies = append(query.Currencies, code)
	}

	for _, text := range in.Banks {
		code, err := domain.ParseBankCode(text)
		if err != nil {
			return app.CommercialRatesQuery{}, unknownBank(text, h.rates.Banks(), err)
		}

		query.Banks = append(query.Banks, code)
	}

	return query, nil
}

// commercialError gives the model a message that names the cause and the next action.
func (h commercialHandler) commercialError(query app.CommercialRatesQuery, err error) error {
	if countErr, ok := errors.AsType[*app.CurrencyCountError](err); ok {
		return currencyCount(countErr.Count, err)
	}

	if unknown, ok := errors.AsType[*app.UnknownBankError](err); ok {
		return unknownBank(string(unknown.Code), h.rates.Banks(), err)
	}

	if errors.Is(err, kernel.ErrSourceUnavailable) {
		return &mcpserver.ToolError{
			Message: joinNames(h.requested(query)) + " did not answer. Try again in a few minutes.",
			Cause:   err,
		}
	}

	return &mcpserver.ToolError{
		Message: "The server failed to read the commercial rates.",
		Cause:   err,
	}
}

// requested returns the names of the banks of the query, or of every bank for none.
func (h commercialHandler) requested(query app.CommercialRatesQuery) []string {
	var names []string

	for _, bank := range h.rates.Banks() {
		if len(query.Banks) == 0 || slices.Contains(query.Banks, bank.Code) {
			names = append(names, bank.Name)
		}
	}

	return names
}

func toCommercialOutput(rates app.CommercialRates) commercialRatesOutput {
	out := commercialRatesOutput{
		Meaning: meaning,
		Banks:   make([]bankRatesOutput, 0, len(rates.Banks)),
	}

	for _, bank := range rates.Banks {
		bankOut := bankRatesOutput{
			Bank:   string(bank.Bank.Code),
			Source: bank.Bank.Name,
			AsOf:   instant(bank.ReadAt),
			Stale:  bank.Stale,
			Rates:  make([]commercialRateOutput, 0, len(bank.Rates)),
		}

		for _, rate := range bank.Rates {
			bankOut.Rates = append(bankOut.Rates, commercialRateOutput{
				Currency: string(rate.Currency),
				Buy:      exact(rate.Buy),
				Sell:     exact(rate.Sell),
			})
		}

		for _, code := range bank.Missing {
			bankOut.Missing = append(bankOut.Missing, string(code))
		}

		out.Banks = append(out.Banks, bankOut)
	}

	for _, bank := range rates.Unavailable {
		out.Unavailable = append(
			out.Unavailable,
			bankOutput{Bank: string(bank.Code), Source: bank.Name},
		)
	}

	return out
}

func bankList(banks []domain.Bank) string {
	entries := make([]string, 0, len(banks))
	for _, bank := range banks {
		entries = append(entries, fmt.Sprintf("%s (%s)", bank.Code, bank.Name))
	}

	return strings.Join(entries, ", ")
}

// joinNames joins names as English does: "A", "A and B", "A, B and C".
func joinNames(names []string) string {
	if len(names) <= 1 {
		return strings.Join(names, "")
	}

	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
