package tool

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	// maxListedValues bounds the codes that a message lists.
	maxListedValues = 20

	queryProblem = "query: give 1 to 100 characters."
	monthProblem = "month: give the month as YYYY-MM, for example 2026-10."
	loading      = "The server is still reading the catalog of Geostat. Try again in a few " +
		"minutes. stats_get_inflation answers now."
	unavailable = "Geostat did not answer. Try again in a few minutes."
	failed      = "The server failed to read the data of Geostat."
	searchHint  = "Call stats_search_tables for the codes."
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

func parseCode(text string) (domain.TableCode, error) {
	code, err := domain.ParseTableCode(text)
	if err != nil {
		return domain.TableCode{}, &mcpserver.ToolError{
			Message: fmt.Sprintf("code: %q is not a table code. %s", text, searchHint),
			Cause:   err,
		}
	}

	return code, nil
}

func parseMonth(text string) (*domain.Month, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No month asks the newest month.
	}

	month, err := domain.ParseMonth(text)
	if err != nil {
		return nil, &mcpserver.ToolError{Message: monthProblem, Cause: err}
	}

	return &month, nil
}

func parseRegion(text string) (domain.Region, error) {
	if text == "" {
		return "", nil
	}

	region, err := domain.ParseRegion(text)
	if err != nil {
		return "", &mcpserver.ToolError{
			Message: fmt.Sprintf("region: %q is not a region of the index. Use %s.",
				text, either(domain.Regions())),
			Cause: err,
		}
	}

	return region, nil
}

func parseGroup(text string) (domain.PriceGroup, error) {
	if text == "" {
		return "", nil
	}

	group, err := domain.ParsePriceGroup(text)
	if err != nil {
		return "", &mcpserver.ToolError{
			Message: fmt.Sprintf("group: %q is not a group of the index. Use %s.",
				text, either(domain.PriceGroups())),
			Cause: err,
		}
	}

	return group, nil
}

// either writes the codes as "a, b, or c".
func either[T ~string](codes []T) string {
	texts := make([]string, 0, len(codes))
	for _, code := range codes {
		texts = append(texts, string(code))
	}

	last := len(texts) - 1

	return strings.Join(texts[:last], ", ") + ", or " + texts[last]
}

// toolError gives the model a message that names the cause and the next action. It
// never repeats the text of a Go error or of Geostat.
func toolError(err error, code domain.TableCode) error {
	if message, found := selectionMessage(err); found {
		return &mcpserver.ToolError{Message: message, Cause: err}
	}

	if message, found := monthMessage(err); found {
		return &mcpserver.ToolError{Message: message, Cause: err}
	}

	var message string

	switch {
	case errors.Is(err, app.ErrInvalidQuery):
		message = queryProblem
	case errors.Is(err, app.ErrCatalogLoading):
		message = loading
	case errors.Is(err, app.ErrUnknownTable):
		message = fmt.Sprintf("code: Geostat has no table %q. %s", code, searchHint)
	case errors.Is(err, kernel.ErrSourceUnavailable):
		message = unavailable
	default:
		message = failed
	}

	return &mcpserver.ToolError{Message: message, Cause: err}
}

func selectionMessage(err error) (string, bool) {
	var problem *app.SelectionError
	if !errors.As(err, &problem) {
		return "", false
	}

	switch problem.Problem {
	case app.ProblemUnknownDimension:
		return fmt.Sprintf("selection: the table has no dimension %q. Use %s.",
			problem.Dimension, strings.Join(problem.Accepted, ", ")), true
	case app.ProblemUnknownValue:
		return fmt.Sprintf("selection: the dimension %q has no value %q. %s",
			problem.Dimension, problem.Value, accepted(problem.Accepted)), true
	case app.ProblemValuesOrLast:
		return fmt.Sprintf("selection: give values or last for %q, not both, and last from 1.",
			problem.Dimension), true
	case app.ProblemDuplicate:
		return fmt.Sprintf("selection: the dimension %q appears twice.", problem.Dimension), true
	case app.ProblemTooManyCells:
		return "selection: the selection holds " + strconv.Itoa(problem.Cells) +
			" cells. Select 1000 or fewer: give values or last for more dimensions.", true
	}

	return "", false
}

func accepted(codes []string) string {
	if len(codes) > maxListedValues {
		return fmt.Sprintf("Call stats_get_table for its %d values.", len(codes))
	}

	return "Use " + strings.Join(codes, ", ") + "."
}

func monthMessage(err error) (string, bool) {
	var problem *app.MonthError
	if !errors.As(err, &problem) {
		return "", false
	}

	if problem.Newest.IsZero() {
		return fmt.Sprintf("month: Geostat published no consumer price index for %s.",
			problem.Month), true
	}

	return fmt.Sprintf(
		"month: Geostat published no consumer price index for %s. The newest month is %s.",
		problem.Month, problem.Newest), true
}
