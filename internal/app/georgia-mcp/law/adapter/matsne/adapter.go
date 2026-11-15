package matsne

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	matsneclient "github.com/abgeo/georgia-mcp/pkg/matsne"
)

const (
	dateLayout   = "02/01/2006"
	mainType     = "main"
	targetTitle  = "title"
	targetText   = "text"
	sortNewest   = "signingDate_desc"
	sortOldest   = "signingDate_asc"
	pendingState = "toBeActivated"

	fieldsWithNumber    = 4
	fieldsWithoutNumber = 3
)

var lastDate = regexp.MustCompile(`(\d{2}/\d{2}/\d{4})\s*$`)

var (
	errBrokenField = errors.New("a field that does not parse")
	errNoVersion   = errors.New("no version of the text")
)

// Adapter implements app.ActSource with the client of the Legislative Herald.
type Adapter struct {
	client *matsneclient.Client
	now    func() time.Time
}

var _ app.ActSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *matsneclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Search reads one page of the consolidated acts that match the query.
func (a *Adapter) Search(ctx context.Context, query domain.SearchQuery) (app.SearchPage, error) {
	page, err := a.client.Search(ctx, string(query.Language), searchQuery(query))
	if err != nil {
		return app.SearchPage{}, unavailable(err)
	}

	total, err := strconv.Atoi(page.Total)
	if err != nil {
		return app.SearchPage{}, unavailable(fmt.Errorf("%w: total %q", errBrokenField, page.Total))
	}

	acts := make([]domain.ActSummary, 0, len(page.Items))

	for _, item := range page.Items {
		summary, err := convertItem(item)
		if err != nil {
			return app.SearchPage{}, unavailable(err)
		}

		acts = append(acts, summary)
	}

	return app.SearchPage{Acts: acts, Total: total, ReadAt: a.readAt()}, nil
}

// Titles reads the titles that the source suggests for a Georgian text.
func (a *Adapter) Titles(ctx context.Context, text string) (app.TitleList, error) {
	suggestions, err := a.client.Suggest(ctx, text)
	if err != nil {
		return app.TitleList{}, unavailable(err)
	}

	titles := make([]domain.ActSummary, 0, len(suggestions))

	for _, suggestion := range suggestions {
		id, err := domain.ParseActID(suggestion.ID)
		if err != nil {
			return app.TitleList{}, unavailable(err)
		}

		titles = append(titles, domain.ActSummary{
			ID: id, Title: suggestion.Title, Status: stateOf(suggestion.State),
		})
	}

	return app.TitleList{Titles: titles, ReadAt: a.readAt()}, nil
}

// Act reads one version of an act. The source answers an unknown act and a refusal with
// the same page, so that page gives an error that wraps both app.ErrUnknownAct and
// kernel.ErrSourceUnavailable.
func (a *Adapter) Act(
	ctx context.Context,
	id domain.ActID,
	version *int,
	language kernel.Language,
) (app.ActText, error) {
	publication := -1
	if version != nil {
		publication = *version
	}

	doc, err := a.client.Document(ctx, string(language), id.String(), publication)
	if errors.Is(err, matsneclient.ErrBlocked) {
		return app.ActText{}, fmt.Errorf(
			"reading act %s: %w: %w: %w", id, app.ErrUnknownAct, kernel.ErrSourceUnavailable, err,
		)
	}

	if err != nil {
		return app.ActText{}, unavailable(err)
	}

	if language == kernel.LanguageEnglish && !translated(doc) {
		return app.ActText{}, fmt.Errorf("reading act %s: %w", id, app.ErrNoTranslation)
	}

	act, err := convertDocument(doc, id, language)
	if err != nil {
		return app.ActText{}, unavailable(err)
	}

	return app.ActText{Act: act, ReadAt: a.readAt()}, nil
}

// Status reads the status of the act with the registration code from a search in
// Georgian. No result gives the zero status.
func (a *Adapter) Status(ctx context.Context, registrationCode string) (app.StatusResult, error) {
	page, err := a.client.Search(ctx, string(kernel.LanguageGeorgian), matsneclient.SearchQuery{
		RegistrationCode: registrationCode, Type: mainType,
	})
	if err != nil {
		return app.StatusResult{}, unavailable(err)
	}

	result := app.StatusResult{ReadAt: a.readAt()}
	if len(page.Items) > 0 {
		result.Status = statusOfClass(page.Items[0].Class)
	}

	return result, nil
}

func (a *Adapter) readAt() time.Time {
	return a.now().In(kernel.TbilisiZone())
}

func unavailable(err error) error {
	return fmt.Errorf("reading the Legislative Herald: %w: %w", kernel.ErrSourceUnavailable, err)
}

func searchQuery(query domain.SearchQuery) matsneclient.SearchQuery {
	out := matsneclient.SearchQuery{
		Query:       query.Text,
		QueryTarget: targetTitle,
		Type:        mainType,
		Issuer:      query.Issuer,
		Number:      query.Number,
		Sort:        sortNewest,
		Page:        query.Page,
	}

	if query.InText {
		out.QueryTarget = targetText
	}

	if query.Order == domain.OldestFirst {
		out.Sort = sortOldest
	}

	if query.Type != nil {
		out.Group = groups()[*query.Type]
	}

	if query.Status != nil {
		out.Status = string(*query.Status)
		if *query.Status == domain.StatusPending {
			out.Status = pendingState
		}
	}

	if query.From != nil {
		out.SigningFrom = query.From.Time().Format(dateLayout)
	}

	if query.To != nil {
		out.SigningTo = query.To.Time().Format(dateLayout)
	}

	return out
}

// groups maps each document type to the group of the source.
func groups() map[domain.DocumentType]string {
	return map[domain.DocumentType]string{
		domain.TypeLaw:                         "1000003",
		domain.TypeParliamentResolution:        "1333937",
		domain.TypePresidentialDecree:          "4829717",
		domain.TypeGovernmentResolution:        "458",
		domain.TypeGovernmentDecree:            "1065776",
		domain.TypeEdict:                       "1000005",
		domain.TypeOrder:                       "1000006",
		domain.TypeInternationalTreaty:         "1319499",
		domain.TypeConstitutionalCourtDecision: "1093052",
		domain.TypeCourtAct:                    "1176355",
		domain.TypeSelfGovernmentOrdinance:     "1000008",
		domain.TypeSelfGovernmentDecision:      "1000007",
		domain.TypeRevenueServiceOrder:         "1365174",
		domain.TypeIndividualLegalAct:          "1151616",
		domain.TypeEULaw:                       "3591321",
		domain.TypeDraft:                       "1776650",
	}
}

// convertItem reads the fields by position: the type, the issuer, the number when there
// are four, and the signing date.
func convertItem(item matsneclient.SearchItem) (domain.ActSummary, error) {
	id, err := domain.ParseActID(item.ID)
	if err != nil {
		return domain.ActSummary{}, fmt.Errorf("%w: %w", errBrokenField, err)
	}

	fields := item.Fields
	if len(fields) != fieldsWithNumber && len(fields) != fieldsWithoutNumber {
		return domain.ActSummary{}, fmt.Errorf(
			"%w: act %s has %d fields",
			errBrokenField,
			id,
			len(fields),
		)
	}

	summary := domain.ActSummary{
		ID: id, Title: item.Title, Type: fields[0], Issuer: fields[1],
		Status: statusOfClass(item.Class),
	}

	if len(fields) == fieldsWithNumber {
		summary.Number = fields[2]
	}

	signed, err := parseDate(fields[len(fields)-1])
	if err != nil {
		return domain.ActSummary{}, err
	}

	summary.Signed = signed

	return summary, nil
}

func statusOfClass(class string) domain.ActStatus {
	classes := strings.Fields(class)

	switch {
	case slices.Contains(classes, "panel-success"):
		return domain.StatusActive
	case slices.Contains(classes, "panel-danger"):
		return domain.StatusExpired
	case slices.Contains(classes, "panel-warning"):
		return domain.StatusPending
	default:
		return ""
	}
}

func stateOf(state string) domain.ActStatus {
	switch state {
	case string(domain.StatusActive):
		return domain.StatusActive
	case string(domain.StatusExpired):
		return domain.StatusExpired
	case pendingState:
		return domain.StatusPending
	default:
		return ""
	}
}

func parseDate(text string) (kernel.Date, error) {
	parsed, err := time.ParseInLocation(dateLayout, strings.TrimSpace(text), kernel.TbilisiZone())
	if err != nil {
		return kernel.Date{}, fmt.Errorf("%w: date %q", errBrokenField, text)
	}

	return kernel.Today(parsed), nil
}

// translated tells whether an English page holds an English text: a title with no
// Georgian letter and at least one paragraph.
func translated(doc matsneclient.Document) bool {
	georgian := strings.ContainsFunc(doc.Title, func(char rune) bool {
		return char >= 'ა' && char <= 'ჿ'
	})

	return !georgian && slices.ContainsFunc(doc.Paragraphs, func(p matsneclient.Paragraph) bool {
		return strings.TrimSpace(p.Text) != ""
	})
}

func convertDocument(
	doc matsneclient.Document,
	id domain.ActID,
	language kernel.Language,
) (domain.Act, error) {
	act := domain.Act{ID: id, Title: doc.Title, Language: language}

	var published kernel.Date

	// A broken signing date leaves the date empty, since the page of an act is the only
	// source of its text. A broken date of a search result fails the page.
	for _, row := range doc.Meta {
		switch row.Label {
		case "დოკუმენტის ნომერი", "Document number":
			act.Number = row.Value
		case "დოკუმენტის მიმღები", "Document issuer":
			act.Issuer = row.Value
		case "მიღების თარიღი", "Date of issuing":
			act.Signed, _ = parseDate(row.Value)
		case "დოკუმენტის ტიპი", "Document type":
			act.Type = row.Value
		case "გამოქვეყნების წყარო, თარიღი", "Source and date of publishing":
			published = publicationDate(row.Value)
		case "სარეგისტრაციო კოდი", "Registration code":
			act.RegistrationCode = row.Value
		}
	}

	versions, current, err := convertVersions(doc, published)
	if err != nil {
		return domain.Act{}, err
	}

	act.Versions, act.Version = versions, current

	if language == kernel.LanguageEnglish {
		act.Body = englishBody(doc.Paragraphs)
	} else {
		act.Body = georgianBody(doc.Paragraphs)
	}

	return act, nil
}

// convertVersions returns every version, newest first, and the selected version. A page
// with no list of versions holds an act that was never consolidated: its one version 0
// carries the publication date.
func convertVersions(
	doc matsneclient.Document,
	published kernel.Date,
) ([]domain.Version, domain.Version, error) {
	if len(doc.Publications) == 0 {
		only := domain.Version{Number: 0, Date: published}

		return []domain.Version{only}, only, nil
	}

	versions := make([]domain.Version, 0, len(doc.Publications))

	var (
		current domain.Version
		found   bool
	)

	for _, publication := range doc.Publications {
		number, err := strconv.Atoi(publication.Value)
		if err != nil || number < 0 {
			return nil, domain.Version{}, fmt.Errorf(
				"%w: version %q",
				errBrokenField,
				publication.Value,
			)
		}

		date, err := parseDate(publication.Text)
		if err != nil {
			return nil, domain.Version{}, err
		}

		version := domain.Version{Number: number, Date: date}
		versions = append(versions, version)

		if publication.Value == doc.Selected {
			current, found = version, true
		}
	}

	if !found {
		return nil, domain.Version{}, errNoVersion
	}

	slices.SortStableFunc(versions, func(x, y domain.Version) int { return y.Number - x.Number })

	return versions, current, nil
}

// publicationDate returns the last date of "ვებგვერდი, 13/03/2026" or "LHG, 54,
// 12/10/2010", or the zero date.
func publicationDate(text string) kernel.Date {
	match := lastDate.FindStringSubmatch(text)
	if match == nil {
		return kernel.Date{}
	}

	date, _ := parseDate(match[1])

	return date
}
