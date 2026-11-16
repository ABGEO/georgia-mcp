package tool

import (
	"errors"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func parseMoment(text string) (*time.Time, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No moment asks for the newest grid state.
	}

	moment, err := domain.ParseMoment(text)
	if err != nil {
		return nil, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"moment: %q is not a moment. Use 2026-11-16T13:00 for a time of Tbilisi, "+
					"or RFC 3339 with an offset.",
				text,
			),
			Cause: err,
		}
	}

	return &moment, nil
}

// toolError gives the model a message that names the cause and the next action. It
// never repeats the text of a Go error or of GSE.
func toolError(err error) error {
	var (
		momentError *app.MomentError
		noGridError *app.NoGridError
	)

	switch {
	case errors.As(err, &momentError):
		problem := "is more than 31 days ago"
		if momentError.Future {
			problem = "is in the future"
		}

		return &mcpserver.ToolError{
			Message: fmt.Sprintf("moment: %s %s. Give a moment from %s to now.",
				instant(momentError.Moment), problem,
				instant(momentError.Earliest.Truncate(time.Second))),
			Cause: err,
		}
	case errors.As(err, &noGridError):
		return &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"GSE published no grid state in the 30 minutes up to %s. Try another moment.",
				instant(noGridError.Slot)),
			Cause: err,
		}
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "GSE did not answer. Try again in a few minutes.", Cause: err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the data of GSE.", Cause: err,
		}
	}
}
