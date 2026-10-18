package tool

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func invalidCode(field, text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"%s: %q is not a currency code. Use three letters, for example USD.",
			field,
			text,
		),
		Cause: cause,
	}
}

func tooManyCodes(count int) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"currencies: %d codes. Ask for %d or fewer, or for none to get every currency.",
			count, maxCurrencies,
		),
	}
}

func invalidDate(field, text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf("%s: %q is not a date. Use YYYY-MM-DD.", field, text),
		Cause:   cause,
	}
}

func invalidLanguage(text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf("language: %q is not a language of the NBG. Use en or ka.", text),
		Cause:   cause,
	}
}

func periodReversed(from, to domain.Date, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf("to: %s is before from %s.", to, from),
		Cause:   cause,
	}
}

// toolError gives the model a message that names the cause and the next action. It
// never repeats the text of a Go error or of the vendor.
func toolError(err error) error {
	if notPublished, ok := errors.AsType[*app.NotPublishedError](err); ok {
		return &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"The NBG has not published rates for %s. The newest set is valid from %s.",
				notPublished.Requested, notPublished.Newest,
			),
			Cause: err,
		}
	}

	if tooLong, ok := errors.AsType[*app.PeriodTooLongError](err); ok {
		return &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"to: the period has %d days. The longest period is %d days.",
				tooLong.Days,
				tooLong.Max,
			),
			Cause: err,
		}
	}

	switch {
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "The NBG did not answer. Try again in a few minutes.",
			Cause:   err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the official rates.",
			Cause:   err,
		}
	}
}
