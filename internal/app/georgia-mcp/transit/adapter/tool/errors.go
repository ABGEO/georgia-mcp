package tool

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const queryProblem = "query: give 1 to 100 characters."

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

// checkQuery refuses a text of more than 100 characters, and an empty one when the
// text is required.
func checkQuery(text string, required bool) error {
	length := utf8.RuneCountInString(strings.TrimSpace(text))
	if (required && length == 0) || length > maxQuery {
		return &mcpserver.ToolError{Message: queryProblem}
	}

	return nil
}

func optionalRouteKind(text string) (domain.Kind, error) {
	if text == "" {
		return "", nil
	}

	kind, err := domain.ParseRouteKind(text)
	if err != nil {
		return "", invalidKind("kind", "route", text, domain.RouteKinds(), err)
	}

	return kind, nil
}

func optionalDate(text string) (kernel.Date, error) {
	if text == "" {
		return kernel.Date{}, nil
	}

	date, err := kernel.ParseDate(text)
	if err != nil {
		return kernel.Date{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("date: %q is not a date. Use YYYY-MM-DD.", text),
			Cause:   err,
		}
	}

	return date, nil
}

func toPoint(field string, lat, lon float64) (domain.Point, error) {
	point, err := domain.NewPoint(lat, lon)
	if err != nil {
		return domain.Point{}, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"%s: %s is not a point. Use a latitude of -90 to 90 and a longitude of -180 to 180.",
				field,
				pointText(lat, lon),
			),
			Cause: err,
		}
	}

	return point, nil
}

func invalidKind(field, of, text string, kinds []domain.Kind, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"%s: %q is not a kind of %s. Use one of: %s.",
			field,
			text,
			of,
			kindList(kinds),
		),
		Cause: cause,
	}
}

func invalidTiming(text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"timing: %q is not valid. Use leave_now, depart_at, or arrive_by.",
			text,
		),
		Cause: cause,
	}
}

func missingTime(timing domain.Timing) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf("time: %s needs a time. Use HH:MM.", timing),
		Cause:   domain.ErrInvalidClockTime,
	}
}

func invalidTime(text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf("time: %q is not a time. Use HH:MM.", text),
		Cause:   errors.Join(domain.ErrInvalidClockTime, cause),
	}
}

func invalidPreference(text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"preference: %q is not valid. Use fastest, less_walking, or fewer_transfers.", text,
		),
		Cause: cause,
	}
}

func invalidTripKinds(cause error) error {
	return &mcpserver.ToolError{
		Message: "kinds: the TTC plans bus and minibus together. Give both, or neither.",
		Cause:   cause,
	}
}

// toolError gives the model a message that names the cause and the next action. It
// never repeats the text of a Go error or of the TTC.
func toolError(err error) error {
	if message, ok := domainMessage(err); ok {
		return &mcpserver.ToolError{Message: message, Cause: err}
	}

	switch {
	case errors.Is(err, app.ErrEmptyQuery):
		return &mcpserver.ToolError{Message: queryProblem, Cause: err}
	case errors.Is(err, app.ErrTripNotFound):
		return &mcpserver.ToolError{
			Message: "The TTC found no trip between the two points. Try other points or another time.",
			Cause:   err,
		}
	case errors.Is(err, kernel.ErrSourceUnavailable):
		return &mcpserver.ToolError{
			Message: "The TTC did not answer. Try again in a few minutes.",
			Cause:   err,
		}
	default:
		return &mcpserver.ToolError{
			Message: "The server failed to read the data of the TTC.",
			Cause:   err,
		}
	}
}

// domainMessage returns the message of an error of a use case that carries its facts.
func domainMessage(err error) (string, bool) {
	if unknown, ok := errors.AsType[*app.UnknownStopError](err); ok {
		return fmt.Sprintf(
			"stop: %q is not a stop of the TTC. Use transit_search_stops to find it.", unknown.Text,
		), true
	}

	if unknown, ok := errors.AsType[*app.UnknownRouteError](err); ok {
		return fmt.Sprintf(
			"route: %q is not a route of the TTC. Use transit_search_routes to find it.",
			unknown.Text,
		), true
	}

	if ambiguous, ok := errors.AsType[*app.AmbiguousRouteError](err); ok {
		return ambiguousMessage(ambiguous), true
	}

	if direction, ok := errors.AsType[*app.DirectionError](err); ok {
		return directionMessage(direction), true
	}

	if notOn, ok := errors.AsType[*app.StopNotOnDirectionError](err); ok {
		return fmt.Sprintf(
			"stop: %q is not on direction %d of route %s. Use transit_get_route to see its stops.",
			notOn.Stop.ID, notOn.Direction, notOn.Route.Number,
		), true
	}

	if outside, ok := errors.AsType[*app.DateOutsideTimetableError](err); ok {
		return fmt.Sprintf(
			"date: %s is outside the timetable. Use a date from %s to %s.",
			outside.Date, outside.First, outside.Last,
		), true
	}

	if outside, ok := errors.AsType[*app.OutsideTbilisiError](err); ok {
		field := outside.Field
		if field == "point" {
			field = "lat, lon"
		}

		return fmt.Sprintf(
			"%s: %s is outside Tbilisi. The TTC serves Tbilisi only.",
			field, pointText(outside.Point.Lat, outside.Point.Lon),
		), true
	}

	return "", false
}

func ambiguousMessage(err *app.AmbiguousRouteError) string {
	routes := make([]string, 0, len(err.Routes))
	for _, route := range err.Routes {
		routes = append(routes, fmt.Sprintf(
			"%s (%s, %s)", route.Number, route.Kind, route.Name.In(kernel.LanguageEnglish),
		))
	}

	return fmt.Sprintf(
		"route: %q names %d routes: %s. Give the kind.",
		err.Text,
		len(err.Routes),
		strings.Join(routes, ", "),
	)
}

func directionMessage(err *app.DirectionError) string {
	directions := slices.Clone(err.Directions)
	slices.SortFunc(directions, func(x, y domain.Direction) int { return x.ID - y.ID })

	names := make([]string, 0, len(directions))
	for _, direction := range directions {
		names = append(
			names,
			fmt.Sprintf("%d (%s)", direction.ID, direction.Headsign.In(kernel.LanguageEnglish)),
		)
	}

	return fmt.Sprintf(
		"direction: route %s has %d directions: %s. Give one.",
		err.Route.Number, len(directions), strings.Join(names, ", "),
	)
}

func kindList(kinds []domain.Kind) string {
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}

	return strings.Join(names, ", ")
}

func pointText(lat, lon float64) string {
	return strconv.FormatFloat(lat, 'f', -1, 64) + ", " + strconv.FormatFloat(lon, 'f', -1, 64)
}
