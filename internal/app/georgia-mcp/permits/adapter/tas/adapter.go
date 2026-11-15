// Package tas reads the applications from the TAS client and converts them to the domain.
// It is the anti-corruption layer of the context for TAS.
package tas

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	tasclient "github.com/abgeo/georgia-mcp/pkg/tas"
)

const (
	pageURL     = "https://tas.ge/?p=publicpage&documentId="
	numberLimit = 50
)

var (
	errUnknownStatus = errors.New("an unknown status")
	errNoDate        = errors.New("an application with no date of registration")
)

// Adapter implements app.ApplicationSource with the TAS client.
type Adapter struct {
	client *tasclient.Client
	now    func() time.Time
}

var _ app.ApplicationSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *tasclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Match reads one page of domain.MaxMatches applications, with no decision text.
func (a *Adapter) Match(ctx context.Context, filter app.SearchFilter) (app.MatchRead, error) {
	request := tasclient.Filter{StatusIDs: statusIDs(), Limit: domain.MaxMatches}

	if !filter.Code.IsZero() {
		request.CadastralCode = filter.Code.String()
	}

	if !filter.Period.IsZero() {
		request.From = midnight(filter.Period.From)
		request.To = midnight(filter.Period.To.AddDays(1)).Add(-time.Millisecond)
	}

	page, err := a.client.SearchDocuments(ctx, request)
	if err != nil {
		return app.MatchRead{}, unavailable(err)
	}

	applications := make([]domain.Application, 0, len(page.Documents))

	for _, document := range page.Documents {
		application, err := toApplication(document)
		if err != nil {
			return app.MatchRead{}, unavailable(err)
		}

		application.DecisionText = ""
		applications = append(applications, application)
	}

	return app.MatchRead{Applications: applications, Total: page.Total, ReadAt: a.readAt()}, nil
}

// Application reads the application whose number equals the number. TAS matches a prefix.
func (a *Adapter) Application(
	ctx context.Context,
	number domain.ApplicationNumber,
) (app.ApplicationRead, error) {
	page, err := a.client.SearchDocuments(ctx, tasclient.Filter{
		DocumentNo: number.String(),
		StatusIDs:  statusIDs(),
		Limit:      numberLimit,
	})
	if err != nil {
		return app.ApplicationRead{}, unavailable(err)
	}

	for _, document := range page.Documents {
		if document.DocumentNo != number.String() {
			continue
		}

		application, err := toApplication(document)
		if err != nil {
			return app.ApplicationRead{}, unavailable(err)
		}

		return app.ApplicationRead{Application: application, ReadAt: a.readAt()}, nil
	}

	return app.ApplicationRead{}, fmt.Errorf("searching %s: %w", number, app.ErrApplicationNotFound)
}

func (a *Adapter) readAt() time.Time {
	return a.now().In(kernel.TbilisiZone())
}

func toApplication(document tasclient.Document) (domain.Application, error) {
	number, err := domain.ParseApplicationNumber(document.DocumentNo)
	if err != nil {
		return domain.Application{}, fmt.Errorf("converting the number: %w", err)
	}

	if document.RegistrationDate == nil {
		return domain.Application{}, fmt.Errorf("%w: %s", errNoDate, number)
	}

	status, err := statusOf(document.DocumentStatusID)
	if err != nil {
		return domain.Application{}, err
	}

	info := document.CachedInfo
	kind := info.Nomenclature

	return domain.Application{
		Number:         number,
		URL:            pageURL + strconv.FormatInt(document.DocumentID, 10),
		Address:        clean(document.Address),
		Registered:     kernel.Today(document.RegistrationDate.Time()),
		Deadline:       deadline(document.DeadLineDate),
		Status:         status,
		Category:       clean(kind.CategoryName),
		ParentCategory: clean(kind.ParentCategoryName),
		Class:          classOf(kind.ClassName),
		Stages:         stages(kind.Stadiums),
		Action:         clean(kind.ActionName),
		Executor:       clean(info.ExecutorName),
		DecisionText:   plainText(document.ResponseText),
	}, nil
}

// The status codes of TAS that the page shows.
const (
	tasApproved        = 1
	tasRefused         = 2
	tasInReview        = 5
	tasReviewedAgain   = 6
	tasAnswering       = 9
	tasDeficient       = 7
	tasNotReviewed     = 11
	tasConditional     = 12
	tasAtCommission    = 20
	tasAtAgencies      = 21
	tasAwaitingPayment = 25
	tasAtCulture       = 26
	tasRevoked         = 27
	tasCompleted       = 28
)

// statuses maps each code that the page of TAS sends for its option "all" to a status.
func statuses() map[int]domain.Status {
	return map[int]domain.Status{
		tasApproved:        domain.StatusApproved,
		tasRefused:         domain.StatusRefused,
		tasDeficient:       domain.StatusDeficient,
		tasConditional:     domain.StatusConditional,
		tasNotReviewed:     domain.StatusNotReviewed,
		tasRevoked:         domain.StatusRevoked,
		tasCompleted:       domain.StatusCompleted,
		tasAwaitingPayment: domain.StatusAwaitingPayment,
		tasInReview:        domain.StatusInReview,
		tasReviewedAgain:   domain.StatusInReview,
		tasAnswering:       domain.StatusInReview,
		tasAtCommission:    domain.StatusInReview,
		tasAtAgencies:      domain.StatusInReview,
		tasAtCulture:       domain.StatusInReview,
	}
}

// statusOf maps a code of TAS to a status. Another code means a change of the source.
func statusOf(id int) (domain.Status, error) {
	status, found := statuses()[id]
	if !found {
		return "", unavailable(fmt.Errorf("%w: %d", errUnknownStatus, id))
	}

	return status, nil
}

func statusIDs() []int {
	return slices.Sorted(maps.Keys(statuses()))
}

func midnight(date kernel.Date) time.Time {
	day := date.Time()

	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, kernel.TbilisiZone())
}

func deadline(instant *tasclient.Millis) *kernel.Date {
	if instant == nil {
		return nil
	}

	date := kernel.Today(instant.Time())

	return &date
}

// clean trims the text and joins its runs of spaces. A text of only quotation marks and
// spaces, or a dash, is absent.
func clean(text string) string {
	joined := strings.Join(strings.Fields(text), " ")
	if joined == "-" || strings.Trim(joined, `" `) == "" {
		return ""
	}

	return joined
}

// classOf keeps the first word of the class: "IV კლასი" gives "IV".
func classOf(name string) string {
	first, _, _ := strings.Cut(clean(name), " ")

	return first
}

func stages(names []string) []string {
	var kept []string

	for _, name := range names {
		if stage := clean(name); stage != "" {
			kept = append(kept, stage)
		}
	}

	return kept
}

func unavailable(err error) error {
	return fmt.Errorf("reading TAS: %w: %w", kernel.ErrSourceUnavailable, err)
}
