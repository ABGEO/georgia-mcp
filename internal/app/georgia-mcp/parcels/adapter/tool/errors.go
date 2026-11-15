package tool

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
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

func parseCode(text string) (domain.CadastralCode, error) {
	code, err := domain.ParseCadastralCode(text)
	if err == nil {
		return code, nil
	}

	if unit, ok := errors.AsType[*domain.UnitCodeError](err); ok {
		return domain.CadastralCode{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"code: %q names a unit. This tool covers parcels only. Use the parcel %s.",
				strings.TrimSpace(text), unit.Parcel),
			Cause: err,
		}
	}

	return domain.CadastralCode{}, &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"code: %q is not a cadastral code. Use 4 or 5 groups of digits with dots, as 01.15.05.066.020.",
			strings.TrimSpace(text),
		),
		Cause: err,
	}
}

func parsePoint(lat, lon float64) (domain.Point, error) {
	point, err := domain.NewPoint(lat, lon)
	if err != nil {
		return domain.Point{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"lat, lon: %v, %v is outside Georgia. "+
					"Give the latitude (41.0 to 43.6), then the longitude (40.0 to 46.8).",
				lat, lon),
			Cause: err,
		}
	}

	return point, nil
}

// toolError gives the model a message that names the cause and the next action. It never
// repeats the text of a Go error or of NAPR.
func toolError(err error, code domain.CadastralCode) error {
	switch {
	case errors.Is(err, app.ErrParcelNotFound):
		return &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"code: NAPR has no parcel %s. Find the code with parcels_list_nearby_parcels "+
					"or parcels_search_addresses.", code),
			Cause: err,
		}
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "NAPR did not answer. Try again in a few minutes.", Cause: err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the data of NAPR.",
			Cause:   err,
		}
	}
}
