package tool

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const queryProblem = "query: give 3 to 100 characters."

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

// toolError gives the model a message that names the cause and the next action. It
// never repeats the text of a Go error or of Georgian Post.
func toolError(err error) error {
	switch {
	case errors.Is(err, app.ErrInvalidQuery):
		return &mcpserver.ToolError{Message: queryProblem, Cause: err}
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "Georgian Post did not answer. Try again in a few minutes.",
			Cause:   err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the data of Georgian Post.",
			Cause:   err,
		}
	}
}
