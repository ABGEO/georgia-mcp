package tool

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// defaultDays is the period of one end alone.
const defaultDays = 7

// withinRange returns the default for 0, the value inside 1 to the maximum, or an error.
func withinRange(field string, value, fallback, maximum int) (int, error) {
	if value == 0 {
		return fallback, nil
	}

	if value < 1 || value > maximum {
		return 0, &mcpserver.ToolError{
			Message: fmt.Sprintf("%s: %d is out of range. Use 1 to %d.", field, value, maximum),
		}
	}

	return value, nil
}

func parseFilter(in searchInput) (app.SearchFilter, error) {
	var filter app.SearchFilter

	if strings.TrimSpace(in.Code) != "" {
		code, err := domain.ParseCadastralCode(in.Code)
		if err != nil {
			return app.SearchFilter{}, &mcpserver.ToolError{
				Message: fmt.Sprintf(
					"code: %q is not a cadastral code. "+
						"Use 4 or 5 groups of digits with dots, as 01.18.04.020.005.",
					strings.TrimSpace(in.Code)),
				Cause: err,
			}
		}

		filter.Code = code
	}

	period, err := parsePeriod(in.From, in.To)
	if err != nil {
		return app.SearchFilter{}, err
	}

	filter.Period = period

	if filter.IsZero() {
		return app.SearchFilter{}, &mcpserver.ToolError{
			Message: "code, from, to: empty. Give a cadastral code, a period, or both.",
			Cause:   app.ErrNoFilter,
		}
	}

	return filter, nil
}

// parsePeriod reads the two ends. One end alone gives a period of defaultDays days.
func parsePeriod(fromText, toText string) (domain.Period, error) {
	from, err := optionalDate("from", fromText)
	if err != nil {
		return domain.Period{}, err
	}

	to, err := optionalDate("to", toText)
	if err != nil {
		return domain.Period{}, err
	}

	switch {
	case from.IsZero() && to.IsZero():
		return domain.Period{}, nil
	case to.IsZero():
		to = from.AddDays(defaultDays - 1)
	case from.IsZero():
		from = to.AddDays(1 - defaultDays)
	}

	period, err := domain.NewPeriod(from, to)

	switch {
	case errors.Is(err, domain.ErrInvalidPeriod):
		return domain.Period{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"from, to: %s is after %s. Give the earlier date as from.",
				from,
				to,
			),
			Cause: err,
		}
	case errors.Is(err, domain.ErrPeriodTooLong):
		return domain.Period{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("from, to: %s to %s is %d days. Use at most %d days.",
				from, to, kernel.DaysBetween(from, to)+1, domain.MaxPeriodDays),
			Cause: err,
		}
	}

	return period, err
}

func optionalDate(field, text string) (kernel.Date, error) {
	if strings.TrimSpace(text) == "" {
		return kernel.Date{}, nil
	}

	date, err := kernel.ParseDate(strings.TrimSpace(text))
	if err != nil {
		return kernel.Date{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("%s: %q is not a date. Use YYYY-MM-DD.", field, text),
			Cause:   err,
		}
	}

	return date, nil
}

func parseNumber(text string) (domain.ApplicationNumber, error) {
	number, err := domain.ParseApplicationNumber(text)
	if err != nil {
		return domain.ApplicationNumber{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"number: %q is not an application number. Use AR1 and digits, as AR11168188.",
				strings.TrimSpace(text)),
			Cause: err,
		}
	}

	return number, nil
}

// toolError gives the model a message that names the cause and the next action. It never
// repeats the text of a Go error or of TAS.
func toolError(err error, number domain.ApplicationNumber) error {
	if tooMany, ok := errors.AsType[*app.TooManyMatchesError](err); ok {
		return &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"TAS has %d applications for this search. The server sorts at most %d. "+
					"Give a shorter period, or a cadastral code.", tooMany.Count, domain.MaxMatches),
			Cause: err,
		}
	}

	switch {
	case errors.Is(err, app.ErrApplicationNotFound):
		return &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"number: TAS has no application %s. Find the number with permits_search_applications.",
				number,
			),
			Cause: err,
		}
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "TAS did not answer. Try again in a few minutes.",
			Cause:   err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the data of TAS.",
			Cause:   err,
		}
	}
}
