package tool

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func parseLanguage(text string) (kernel.Language, error) {
	language, err := kernel.ParseLanguage(text)
	if err != nil {
		return "", &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"language: %q is not a language of this tool. Use ka or en.",
				text,
			),
			Cause: err,
		}
	}

	return language, nil
}

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

func parseDate(field, text string) (*kernel.Date, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No date asks for the default.
	}

	date, err := kernel.ParseDate(text)
	if err != nil {
		return nil, &mcpserver.ToolError{
			Message: fmt.Sprintf("%s: %q is not a date. Use YYYY-MM-DD.", field, text),
			Cause:   err,
		}
	}

	return &date, nil
}

// toolError gives the model a message that names the cause and the next action. field
// names the input that held the date of a *app.YearError, or "year". It never repeats the
// text of a Go error or of Nager.Date.
func toolError(err error, field, date string) error {
	var yearError *app.YearError

	switch {
	case errors.Is(err, app.ErrBothFilters):
		return &mcpserver.ToolError{
			Message: "year, from: give a year or a from date, not both.", Cause: err,
		}
	case errors.As(err, &yearError) && field == "year":
		return &mcpserver.ToolError{
			Message: fmt.Sprintf("year: %d is not supported. Use %d to %d.",
				yearError.Year, yearError.First, yearError.Last),
			Cause: err,
		}
	case errors.As(err, &yearError):
		return &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"%s: %s is not in a supported year. Use a date from %d-01-01 to %d-12-31.",
				field, date, yearError.First, yearError.Last),
			Cause: err,
		}
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "Nager.Date did not answer. Try again in a few minutes.", Cause: err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the data of Nager.Date.", Cause: err,
		}
	}
}
