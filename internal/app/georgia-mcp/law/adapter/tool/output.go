package tool

import (
	"fmt"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const translationNotice = "This is an English translation. Only the Georgian text is official."

func toSearchOutput(result app.SearchResult, query domain.SearchQuery) searchOutput {
	acts := make([]actSummary, 0, len(result.Acts))

	for _, act := range result.Acts {
		acts = append(acts, actSummary{
			ID:     act.ID.String(),
			Title:  act.Title,
			Type:   act.Type,
			Issuer: act.Issuer,
			Number: act.Number,
			Signed: day(act.Signed),
			Status: string(act.Status),
			URL:    actURL(query.Language, act.ID),
		})
	}

	return searchOutput{
		Source: source,
		AsOf:   instant(result.ReadAt),
		Stale:  result.Stale,
		Total:  result.Total,
		Page:   query.Page,
		Acts:   acts,
	}
}

func toActOutput(view app.ActView, language kernel.Language) actOutput {
	act := view.Act

	out := actOutput{
		Source: source,
		AsOf:   instant(view.ReadAt),
		Stale:  view.Stale,
		URL: fmt.Sprintf(
			"%s?publication=%d",
			actURL(language, act.ID),
			act.Version.Number,
		),
		Language:     string(language),
		Official:     language == kernel.LanguageGeorgian,
		ID:           act.ID.String(),
		Title:        act.Title,
		Type:         act.Type,
		Issuer:       act.Issuer,
		Number:       act.Number,
		Signed:       day(act.Signed),
		Status:       string(view.Status),
		Registration: act.RegistrationCode,
		Version:      toVersion(act.Version),
		Truncated:    view.Truncated,
		Text:         view.Text,
		Piece:        view.Piece,
		Pieces:       view.Pieces,
	}

	out.Notice = notice(view, out.Official)

	if view.Newest != nil {
		newest := toVersion(*view.Newest)
		out.NewestVersion = &newest
	}

	if view.Next != nil {
		out.Next = view.Next.String()
	}

	addParts(&out, view)

	return out
}

// addParts adds the versions, the contents, and the articles of the view.
func addParts(out *actOutput, view app.ActView) {
	for _, version := range view.Act.Versions {
		out.Versions = append(out.Versions, toVersion(version))
	}

	if out.Versions != nil {
		out.VersionCount = view.VersionCount
	}

	for _, heading := range view.Contents {
		out.Contents = append(out.Contents, headingOutput{
			Kind: string(heading.Kind), Number: heading.Number, Title: heading.Title,
		})
	}

	for _, article := range view.Articles {
		out.Articles = append(out.Articles, articleOutput{
			Number: article.Number.String(), Title: article.Title, Paragraphs: article.Paragraphs,
		})
	}
}

// notice tells that a text is a translation, and names the newest version when the
// translation trails it.
func notice(view app.ActView, official bool) string {
	var sentences []string

	if !official {
		sentences = append(sentences, translationNotice)
	}

	if view.Newest != nil {
		sentences = append(sentences, fmt.Sprintf(
			"The newest version is %d of %s. Ask for language ka to read it.",
			view.Newest.Number, day(view.Newest.Date),
		))
	}

	return strings.Join(sentences, " ")
}

func toVersion(version domain.Version) versionOutput {
	return versionOutput{Number: version.Number, Date: day(version.Date)}
}

func actURL(language kernel.Language, id domain.ActID) string {
	return siteURL + "/" + string(language) + "/document/view/" + id.String()
}

func day(date kernel.Date) string {
	if date.IsZero() {
		return ""
	}

	return date.String()
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
