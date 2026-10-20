package tool

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func toolError(message string, cause error) error {
	return &mcpserver.ToolError{Message: message, Cause: cause}
}

func invalidBankCode(field, text string, banks []domain.Bank, cause error) error {
	return toolError(
		fmt.Sprintf("%s: %q is not a bank code. Use one of: %s.", field, text, codeList(banks)),
		cause,
	)
}

func invalidValue(field, text, noun, accepted string, cause error) error {
	return toolError(
		fmt.Sprintf("%s: %q is not %s. Use one of: %s.", field, text, noun, accepted),
		cause,
	)
}

func invalidCardCode(text string, cause error) error {
	return toolError(
		fmt.Sprintf(
			"cards: %q is not a card code. Use the lowercase codes of the description.",
			text,
		),
		cause,
	)
}

func invalidDate(field, text string, cause error) error {
	return toolError(fmt.Sprintf("%s: %q is not a date. Use YYYY-MM-DD.", field, text), cause)
}

func invalidLanguage(text string, cause error) error {
	return toolError(
		fmt.Sprintf("language: %q is not a language of this tool. Use ka or en.", text),
		cause,
	)
}

func invalidLimit(limit int) error {
	return toolError(
		fmt.Sprintf("limit: %d is out of range. Use %d to %d.", limit, minLimit, maxLimit),
		nil,
	)
}

func invalidOffset(offset int) error {
	return toolError(fmt.Sprintf("offset: %d is out of range. Use 0 or more.", offset), nil)
}

func invalidQuery() error {
	return toolError(fmt.Sprintf("query: use 1 to %d characters.", maxQueryLength), nil)
}

func emptyID() error {
	return toolError("id: give the offer id from offers_search_offers.", nil)
}

// failure gives the model a message that names the cause and the next action. It never
// repeats the text of a Go error or of the vendor. field names the input of the bank.
func failure(field string, requested, banks []domain.Bank, id string, err error) error {
	if unknown, ok := errors.AsType[*app.UnknownBankError](err); ok {
		return toolError(fmt.Sprintf(
			"%s: %q is not a bank of this server. Use one of: %s.",
			field,
			unknown.Code,
			codeList(banks),
		), err)
	}

	switch {
	case errors.Is(err, app.ErrInvalidRange):
		return toolError(
			fmt.Sprintf("to: must be from 0 to %d days after from.", maxRangeDays),
			err,
		)
	case errors.Is(err, app.ErrOfferNotFound):
		return toolError(fmt.Sprintf(
			"id: %s lists no current offer %q. Use offers_search_offers.", joinNames(requested), id,
		), err)
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return toolError(joinNames(requested)+" did not answer. Try again in a few minutes.", err)
	}

	return toolError("The server failed to read the offers.", err)
}

func codeList(banks []domain.Bank) string {
	codes := make([]string, 0, len(banks))
	for _, bank := range banks {
		codes = append(codes, string(bank.Code))
	}

	return strings.Join(codes, ", ")
}

// joinNames joins the names of banks as English does: "A", "A and B", "A, B and C".
func joinNames(banks []domain.Bank) string {
	names := make([]string, 0, len(banks))
	for _, bank := range banks {
		names = append(names, bank.Name)
	}

	if len(names) <= 1 {
		return strings.Join(names, "")
	}

	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
