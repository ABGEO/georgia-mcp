package tool

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

func invalidGrade(text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf("grades: %q is not a grade. Use one of: %s.", text, gradeList()),
		Cause:   cause,
	}
}

func invalidCompanyCode(text string, companies []domain.Company, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"companies: %q is not a company code. Use one of: %s.", text, codeList(companies),
		),
		Cause: cause,
	}
}

func unknownCompany(text string, companies []domain.Company, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf(
			"companies: %q is not a fuel company of this server. Use one of: %s.",
			text, codeList(companies),
		),
		Cause: cause,
	}
}

func invalidLanguage(text string, cause error) error {
	return &mcpserver.ToolError{
		Message: fmt.Sprintf("language: %q is not a language of this tool. Use ka or en.", text),
		Cause:   cause,
	}
}

// toolError gives the model a message that names the cause and the next action. It
// never repeats the text of a Go error or of the vendor.
func (h handler) toolError(query app.PricesQuery, err error) error {
	if unknown, ok := errors.AsType[*app.UnknownCompanyError](err); ok {
		return unknownCompany(string(unknown.Code), h.prices.Companies(), err)
	}

	if errors.Is(err, kernel.ErrSourceUnavailable) {
		return &mcpserver.ToolError{
			Message: joinNames(h.requested(query)) + " did not answer. Try again in a few minutes.",
			Cause:   err,
		}
	}

	return &mcpserver.ToolError{
		Message: "The server failed to read the fuel prices.",
		Cause:   err,
	}
}

// requested returns the names of the companies of the query, or of every company for none.
func (h handler) requested(query app.PricesQuery) []string {
	var names []string

	for _, company := range h.prices.Companies() {
		if len(query.Companies) == 0 || slices.Contains(query.Companies, company.Code) {
			names = append(names, company.Name)
		}
	}

	return names
}

func gradeList() string {
	names := make([]string, 0, len(domain.Grades()))
	for _, grade := range domain.Grades() {
		names = append(names, string(grade))
	}

	return strings.Join(names, ", ")
}

func codeList(companies []domain.Company) string {
	codes := make([]string, 0, len(companies))
	for _, company := range companies {
		codes = append(codes, string(company.Code))
	}

	return strings.Join(codes, ", ")
}

// joinNames joins names as English does: "A", "A and B", "A, B and C".
func joinNames(names []string) string {
	if len(names) <= 1 {
		return strings.Join(names, "")
	}

	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
