package tool

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
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

func parseStation(text string) (domain.StationCode, error) {
	code, err := domain.ParseStationCode(text)
	if err != nil {
		return domain.StationCode{}, stationError(text, err)
	}

	return code, nil
}

func parsePollutant(text string) (*domain.Pollutant, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No pollutant asks for every pollutant.
	}

	pollutant, err := domain.ParsePollutant(text)
	if err != nil {
		return nil, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"pollutant: %q is not a pollutant of the NEA. Use PM10, PM2.5, NO2, O3, SO2, or CO.",
				text,
			),
			Cause: err,
		}
	}

	return &pollutant, nil
}

func parseDate(field, text string) (*kernel.Date, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No date asks for the default period.
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

func stationError(text string, err error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"station: %q is not a station of the NEA. Call air_get_quality with a city to find the codes.",
			text,
		),
		Cause: err,
	}
}

// toolError gives the model a message that names the cause and the next action. It
// never repeats the text of a Go error or of the NEA.
func toolError(err error, city, station string) error {
	var (
		cityError   *app.CityError
		periodError *app.PeriodError
	)

	switch {
	case errors.Is(err, app.ErrBothFilters):
		return &mcpserver.ToolError{
			Message: "city, station: give a city or a station, not both.", Cause: err,
		}
	case errors.As(err, &cityError):
		names := make([]string, 0, len(cityError.Accepted))
		for _, accepted := range cityError.Accepted {
			names = append(names, accepted.Name.English)
		}

		return &mcpserver.ToolError{
			Message: fmt.Sprintf("city: %q has no station of the NEA. Use one of: %s.",
				city, strings.Join(names, ", ")),
			Cause: err,
		}
	case errors.Is(err, app.ErrUnknownStation):
		return stationError(station, err)
	case errors.As(err, &periodError):
		return &mcpserver.ToolError{Message: periodError.Message, Cause: err}
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "The NEA did not answer. Try again in a few minutes.", Cause: err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the data of the NEA.", Cause: err,
		}
	}
}
