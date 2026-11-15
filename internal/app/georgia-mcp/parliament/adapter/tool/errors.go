package tool

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	defaultLimit = 10
	firstPage    = 1
)

// parsePage returns the page and the limit, with the default for 0.
func parsePage(number, limit int) (domain.Page, error) {
	if number == 0 {
		number = firstPage
	}

	if number < firstPage {
		return domain.Page{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("page: %d is out of range. Use 1 or more.", number),
		}
	}

	if limit == 0 {
		limit = defaultLimit
	}

	if limit < 1 || limit > domain.MaxPageSize {
		return domain.Page{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"limit: %d is out of range. Use 1 to %d.",
				limit,
				domain.MaxPageSize,
			),
		}
	}

	return domain.Page{Number: number, Size: limit}, nil
}

// parseText returns the zero text for an empty input.
func parseText(field, value string) (domain.SearchText, error) {
	if value == "" {
		return domain.SearchText{}, nil
	}

	text, err := domain.NewSearchText(value)
	if err != nil {
		return domain.SearchText{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("%s: use 1 to %d characters.", field, domain.MaxTextLength),
			Cause:   err,
		}
	}

	return text, nil
}

func parsePeriod(fromText, toText string) (domain.Period, error) {
	from, err := optionalDate("from", fromText)
	if err != nil {
		return domain.Period{}, err
	}

	to, err := optionalDate("to", toText)
	if err != nil {
		return domain.Period{}, err
	}

	period, err := domain.NewPeriod(from, to)
	if err != nil {
		return domain.Period{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("from, to: %s is after %s.", from, to),
			Cause:   err,
		}
	}

	return period, nil
}

func optionalDate(field, text string) (*kernel.Date, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil //nolint:nilnil // An absent end is not an error.
	}

	date, err := kernel.ParseDate(strings.TrimSpace(text))
	if err != nil {
		return nil, &mcpserver.ToolError{
			Message: fmt.Sprintf("%s: %q is not a date. Use YYYY-MM-DD.", field, text),
			Cause:   err,
		}
	}

	return &date, nil
}

func parseID(value int) (domain.ID, error) {
	id, err := domain.NewID(value)
	if err != nil {
		return 0, &mcpserver.ToolError{
			Message: fmt.Sprintf("id: %d is not an identifier. Use a positive number.", value),
			Cause:   err,
		}
	}

	return id, nil
}

// toolError gives the model a message that names the cause and the next action. It never
// repeats the text of a Go error or of the Parliament. inputs names the criteria of the
// tool for ErrNoCriterion.
func toolError(err error, id domain.ID, inputs string) error {
	message := "The server failed to read the data of the Parliament of Georgia."

	switch {
	case errors.Is(err, app.ErrNoCriterion):
		message = "Give at least one of " + inputs + "."
	case errors.Is(err, app.ErrBillNotFound):
		message = fmt.Sprintf(
			"The Parliament has no bill %d. Find a bill with parliament_search_bills.", id,
		)
	case errors.Is(err, app.ErrVoteNotFound):
		message = fmt.Sprintf("The Parliament has no vote %d. Find a vote with "+
			"parliament_list_votes or parliament_get_bill.", id)
	case errors.Is(err, kernel.ErrSourceUnavailable):
		message = "The Parliament of Georgia did not answer. Try again in a few minutes."
	}

	return &mcpserver.ToolError{Message: message, Cause: err}
}
