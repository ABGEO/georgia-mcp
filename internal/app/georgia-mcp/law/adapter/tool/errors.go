package tool

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
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

func parsePage(page int) (int, error) {
	if page == 0 {
		return 1, nil
	}

	if page < 1 {
		return 0, &mcpserver.ToolError{
			Message: fmt.Sprintf("page: %d is out of range. Use 1 or more.", page),
		}
	}

	return page, nil
}

func parseTarget(text string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "", "title":
		return false, nil
	case "text":
		return true, nil
	default:
		return false, &mcpserver.ToolError{
			Message: fmt.Sprintf("target: %q is not a target. Use title or text.", text),
		}
	}
}

func parseOrder(text string) (domain.Order, error) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "", "newest":
		return domain.NewestFirst, nil
	case "oldest":
		return domain.OldestFirst, nil
	default:
		return 0, &mcpserver.ToolError{
			Message: fmt.Sprintf("order: %q is not an order. Use newest or oldest.", text),
		}
	}
}

func parseType(text string) (*domain.DocumentType, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No type asks for every type.
	}

	docType, err := domain.ParseDocumentType(text)
	if err != nil {
		codes := make([]string, 0, len(domain.DocumentTypes()))
		for _, code := range domain.DocumentTypes() {
			codes = append(codes, string(code))
		}

		return nil, &mcpserver.ToolError{
			Message: fmt.Sprintf("type: %q is not a document type. Use one of: %s.",
				text, strings.Join(codes, ", ")),
			Cause: err,
		}
	}

	return &docType, nil
}

func parseStatus(text string) (*domain.ActStatus, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No status asks for every status.
	}

	status, err := domain.ParseActStatus(text)
	if err != nil {
		return nil, &mcpserver.ToolError{
			Message: fmt.Sprintf(
				"status: %q is not a status. Use active, expired, or pending.",
				text,
			),
			Cause: err,
		}
	}

	return &status, nil
}

func parseDate(field, text string) (*kernel.Date, error) {
	if text == "" {
		return nil, nil //nolint:nilnil // No date asks for no bound.
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

func parseArticles(texts []string) ([]domain.ArticleNumber, error) {
	numbers := make([]domain.ArticleNumber, 0, len(texts))

	for _, text := range texts {
		number, err := domain.ParseArticleNumber(text)
		if err != nil {
			return nil, &mcpserver.ToolError{
				Message: fmt.Sprintf(
					"articles: %q is not an article number. Use digits, with ^ and "+
						"digits for an inserted article, for example 133^3.",
					text,
				),
				Cause: err,
			}
		}

		numbers = append(numbers, number)
	}

	return numbers, nil
}

// toolError gives the model a message that names the cause and the next action. It never
// repeats the text of a Go error or of the source. An unknown act comes before an outage,
// since the source answers both with one page.
func toolError(err error, id string, articles int) error {
	var (
		articleErr *app.ArticleError
		versionErr *app.VersionError
		pieceErr   *app.PieceError
	)

	message := "The server failed to read the data of the Legislative Herald."

	switch {
	case errors.Is(err, app.ErrBothVersions):
		message = "version, date: give a version or a date, not both."
	case errors.Is(err, app.ErrTooManyArticles):
		message = fmt.Sprintf("articles: %d articles. Give at most %d.", articles, app.MaxArticles)
	case errors.As(err, &articleErr):
		message = articleMessage(articleErr)
	case errors.As(err, &versionErr):
		message = fmt.Sprintf(
			"version: the act has versions %d (%s) to %d (%s).",
			versionErr.First.Number,
			versionErr.First.Date,
			versionErr.Newest.Number,
			versionErr.Newest.Date,
		)
	case errors.As(err, &pieceErr):
		message = fmt.Sprintf(
			"piece: the text has %d %s.",
			pieceErr.Pieces,
			plural(pieceErr.Pieces, "piece"),
		)
	case errors.Is(err, app.ErrNoTranslation):
		message = "The act has no English translation of this version at the Legislative Herald. " +
			"Ask for language ka."
	case errors.Is(err, app.ErrUnknownAct):
		message = fmt.Sprintf(
			"id: the Legislative Herald returned no act %s. Check the identifier "+
				"with law_search_acts, or try again in a few minutes.",
			id,
		)
	case errors.Is(err, kernel.ErrSourceUnavailable):
		message = "The Legislative Herald did not answer. Try again in a few minutes."
	}

	return &mcpserver.ToolError{Message: message, Cause: err}
}

func articleMessage(err *app.ArticleError) string {
	if err.First.IsZero() {
		return fmt.Sprintf("articles: the act has no article %s. It has no articles: ask with no "+
			"article to read its text.", err.Number)
	}

	return fmt.Sprintf("articles: the act has no article %s. Its articles run from %s to %s.",
		err.Number, err.First, err.Last)
}

func plural(count int, word string) string {
	if count == 1 {
		return word
	}

	return word + "s"
}
