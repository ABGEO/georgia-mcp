// Package parliament reads the bills, the votes, and the member questions from the client
// of the Parliament and converts them to the domain. It is the anti-corruption layer of
// the context.
package parliament

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	parliamentclient "github.com/abgeo/georgia-mcp/pkg/parliament"
)

const (
	billPage     = parliamentclient.DefaultBaseURL + "/#law-drafting/"
	questionPage = parliamentclient.DefaultBaseURL + "/#mpqs/"
	sourceDate   = "02-01-2006"
	stageVotes   = 10
	// maxMembers is above the 150 seats of the Parliament, so one page holds every choice.
	maxMembers = 200
)

var errNoChoice = errors.New("a vote with no choice")

// Adapter implements the three ports with the client of the Parliament.
type Adapter struct {
	client *parliamentclient.Client
	now    func() time.Time
}

var (
	_ app.BillSource     = (*Adapter)(nil)
	_ app.VoteSource     = (*Adapter)(nil)
	_ app.QuestionSource = (*Adapter)(nil)
)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *parliamentclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// SearchBills reads one page of the bills that match the filter.
func (a *Adapter) SearchBills(
	ctx context.Context,
	filter app.BillFilter,
	page domain.Page,
) (app.BillPage, error) {
	from, to := sourceRange(filter.Registered)

	list, err := a.client.Bills(ctx, parliamentclient.BillQuery{
		Start:              page.Start(),
		Limit:              page.Size,
		BillName:           filter.Text.String(),
		RegistrationNumber: filter.Number,
		RegistrationDate1:  from,
		RegistrationDate2:  to,
	})
	if err != nil {
		return app.BillPage{}, unavailable(err)
	}

	bills := make([]domain.Bill, 0, len(list.List))

	for _, item := range list.List {
		bill, err := toBill(item)
		if err != nil {
			return app.BillPage{}, unavailable(err)
		}

		bills = append(bills, bill)
	}

	return app.BillPage{Bills: bills, Total: list.Total, ReadAt: a.readAt()}, nil
}

// Bill reads the bill and its stages at the same time, then the votes of each stage that
// has counts at the same time.
func (a *Adapter) Bill(ctx context.Context, id domain.ID) (app.BillRead, error) {
	var (
		group               sync.WaitGroup
		item                *parliamentclient.Bill
		reviews             []parliamentclient.BillReview
		billErr, reviewsErr error
	)

	group.Go(func() { item, billErr = a.client.Bill(ctx, int(id)) })
	group.Go(func() { reviews, reviewsErr = a.client.BillReviews(ctx, int(id)) })
	group.Wait()

	switch {
	case billErr != nil:
		return app.BillRead{}, unavailable(billErr)
	case item == nil:
		return app.BillRead{}, fmt.Errorf("reading the bill %d: %w", id, app.ErrBillNotFound)
	case reviewsErr != nil:
		return app.BillRead{}, unavailable(reviewsErr)
	}

	bill, err := toBill(*item)
	if err != nil {
		return app.BillRead{}, unavailable(err)
	}

	stages, err := a.toStages(ctx, reviews)
	if err != nil {
		return app.BillRead{}, unavailable(err)
	}

	return app.BillRead{Bill: bill, Stages: stages, ReadAt: a.readAt()}, nil
}

// ListVotes reads one page of the votes that match the filter.
func (a *Adapter) ListVotes(
	ctx context.Context,
	filter app.VoteFilter,
	page domain.Page,
) (app.VotePage, error) {
	from, to := sourceRange(filter.Voted)

	list, err := a.client.VoteSessions(ctx, parliamentclient.VoteSessionQuery{
		Start:       page.Start(),
		Limit:       page.Size,
		BillRd:      filter.BillTitle.String(),
		VotingDate1: from,
		VotingDate2: to,
	})
	if err != nil {
		return app.VotePage{}, unavailable(err)
	}

	votes := make([]domain.Vote, 0, len(list.List))

	for _, item := range list.List {
		vote, err := toVote(item)
		if err != nil {
			return app.VotePage{}, unavailable(err)
		}

		votes = append(votes, vote)
	}

	return app.VotePage{Votes: votes, Total: list.Total, ReadAt: a.readAt()}, nil
}

// Vote reads the vote and the choice of each member.
func (a *Adapter) Vote(ctx context.Context, id domain.ID) (app.VoteRead, error) {
	item, err := a.client.VoteSession(ctx, int(id))
	if err != nil {
		return app.VoteRead{}, unavailable(err)
	}

	if item == nil {
		return app.VoteRead{}, fmt.Errorf("reading the vote %d: %w", id, app.ErrVoteNotFound)
	}

	vote, err := toVote(*item)
	if err != nil {
		return app.VoteRead{}, unavailable(err)
	}

	list, err := a.client.Votings(ctx, int(id), 0, maxMembers)
	if err != nil {
		return app.VoteRead{}, unavailable(err)
	}

	if len(list.List) == 0 {
		return app.VoteRead{}, unavailable(fmt.Errorf("%w: %d", errNoChoice, id))
	}

	choices := make([]domain.MemberChoice, 0, len(list.List))

	for _, voting := range list.List {
		choices = append(choices, toChoice(voting))
	}

	return app.VoteRead{Vote: vote, Choices: choices, ReadAt: a.readAt()}, nil
}

// SearchQuestions reads one page of the member questions that match the filter.
func (a *Adapter) SearchQuestions(
	ctx context.Context,
	filter app.QuestionFilter,
	page domain.Page,
) (app.QuestionPage, error) {
	from, to := sourceRange(filter.Registered)

	list, err := a.client.MPQuestions(ctx, parliamentclient.MPQuestionQuery{
		Start:             page.Start(),
		Limit:             page.Size,
		RequestTitle:      filter.Text.String(),
		ColleagueName:     filter.Member.String(),
		RegistrationDate1: from,
		RegistrationDate2: to,
	})
	if err != nil {
		return app.QuestionPage{}, unavailable(err)
	}

	questions := make([]domain.Question, 0, len(list.List))

	for _, item := range list.List {
		question, err := toQuestion(item)
		if err != nil {
			return app.QuestionPage{}, unavailable(err)
		}

		questions = append(questions, question)
	}

	return app.QuestionPage{Questions: questions, Total: list.Total, ReadAt: a.readAt()}, nil
}

func (a *Adapter) readAt() time.Time {
	return a.now().In(kernel.TbilisiZone())
}

func (a *Adapter) toStages(
	ctx context.Context,
	reviews []parliamentclient.BillReview,
) ([]domain.Stage, error) {
	stages := make([]domain.Stage, len(reviews))

	for index, review := range reviews {
		stage, err := toStage(review)
		if err != nil {
			return nil, err
		}

		stages[index] = stage
	}

	group, groupCtx := errgroup.WithContext(ctx)

	for index, review := range reviews {
		if review.VoteY == nil && review.VoteN == nil {
			continue
		}

		group.Go(func() error {
			votes, err := a.stageVotes(groupCtx, review.ID)
			stages[index].Votes = votes

			return err
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("reading the votes of the stages: %w", err)
	}

	return stages, nil
}

func (a *Adapter) stageVotes(ctx context.Context, reviewID int) ([]domain.ID, error) {
	list, err := a.client.VoteSessions(ctx, parliamentclient.VoteSessionQuery{
		Limit: stageVotes, BillReviewID: reviewID,
	})
	if err != nil {
		return nil, fmt.Errorf("reading the votes of the stage %d: %w", reviewID, err)
	}

	votes := make([]domain.ID, 0, len(list.List))

	for _, vote := range list.List {
		id, err := domain.NewID(vote.ID)
		if err != nil {
			return nil, fmt.Errorf("converting a vote of the stage %d: %w", reviewID, err)
		}

		votes = append(votes, id)
	}

	return votes, nil
}

func toStage(review parliamentclient.BillReview) (domain.Stage, error) {
	stage := domain.Stage{
		Name:          strings.TrimSpace(review.ReviewTypeName),
		SessionNumber: text(review.SessionNumber),
		BureauNumber:  text(review.BureauNumber),
		Yes:           review.VoteY,
		No:            review.VoteN,
	}

	if reading, found := domain.ReadingOf(review.ReviewTypeName); found {
		stage.Reading = reading
	}

	dates := []struct {
		source *string
		target **kernel.Date
	}{
		{review.BillReviewFromDate, &stage.From},
		{review.BillReviewToDate, &stage.To},
		{review.SessionReviewDate, &stage.SessionDate},
		{review.BureauReviewDate, &stage.BureauDate},
	}

	for _, item := range dates {
		parsed, err := optionalDate(item.source)
		if err != nil {
			return domain.Stage{}, err
		}

		*item.target = parsed
	}

	return stage, nil
}

func toBill(item parliamentclient.Bill) (domain.Bill, error) {
	id, err := domain.NewID(item.ID)
	if err != nil {
		return domain.Bill{}, fmt.Errorf("converting a bill: %w", err)
	}

	registered, err := sourceDateOf(item.BillPackage.RegistrationDate)
	if err != nil {
		return domain.Bill{}, fmt.Errorf("converting the bill %d: %w", id, err)
	}

	bill := domain.Bill{
		ID:                 id,
		Title:              strings.TrimSpace(item.BillName),
		Description:        text(item.BillDescription),
		Type:               strings.TrimSpace(item.BillType.Name),
		RegistrationNumber: strings.TrimSpace(item.BillPackage.RegistrationNumber),
		PackageName:        strings.TrimSpace(item.BillPackage.PackageName),
		Initiator:          strings.TrimSpace(item.BillPackage.Initiator),
		Author:             strings.TrimSpace(item.BillPackage.Author),
		Registered:         registered,
		URL:                billPage + strconv.Itoa(item.ID),
	}

	if count := len(item.BillReviewTypes); count > 0 {
		bill.NewestStage = strings.TrimSpace(item.BillReviewTypes[count-1].Name)

		if reading, found := domain.ReadingOf(bill.NewestStage); found {
			bill.NewestReading = reading
		}
	}

	return bill, nil
}

func toVote(item parliamentclient.VoteSession) (domain.Vote, error) {
	id, err := domain.NewID(item.ID)
	if err != nil {
		return domain.Vote{}, fmt.Errorf("converting a vote: %w", err)
	}

	billID, err := domain.NewID(item.BillIDRd)
	if err != nil {
		return domain.Vote{}, fmt.Errorf("converting the bill of the vote %d: %w", id, err)
	}

	voted, err := sourceDateOf(item.ImportConferenceDate)
	if err != nil {
		return domain.Vote{}, fmt.Errorf("converting the vote %d: %w", id, err)
	}

	readingName := strings.TrimSpace(item.BillReviewTypeRd)

	reading, found := domain.ReadingOf(readingName)
	if !found {
		reading = domain.OtherReading
	}

	outcomeName := ""
	if item.VotingConclusionType != nil {
		outcomeName = strings.TrimSpace(item.VotingConclusionType.Name)
	}

	return domain.Vote{
		ID:                 id,
		Date:               voted,
		BillID:             billID,
		BillTitle:          strings.TrimSpace(item.BillRd),
		RegistrationNumber: strings.TrimSpace(item.RegistrationNumberRd),
		Reading:            domain.Coded[domain.Reading]{Code: reading, Name: readingName},
		Outcome: domain.Coded[domain.Outcome]{
			Code: domain.OutcomeOf(outcomeName), Name: outcomeName,
		},
		Quorum: item.Quorum,
		Yes:    item.VotingResultSet.Yes,
		No:     item.VotingResultSet.No,
		Absent: item.VotingResultSet.WithoutInteraction,
		URL:    billPage + strconv.Itoa(item.BillIDRd),
	}, nil
}

func toChoice(voting parliamentclient.Voting) domain.MemberChoice {
	name := ""
	if voting.VotingResult != nil {
		name = strings.TrimSpace(voting.VotingResult.Name)
	}

	return domain.MemberChoice{
		Member:  strings.TrimSpace(voting.ColleagueRd),
		Faction: text(voting.FactionRd),
		Choice:  domain.Coded[domain.Choice]{Code: domain.ChoiceOf(name), Name: name},
	}
}

func toQuestion(item parliamentclient.MPQuestion) (domain.Question, error) {
	registered, err := sourceDateOf(item.RegistrationDate)
	if err != nil {
		return domain.Question{}, fmt.Errorf("converting the question %d: %w", item.ID, err)
	}

	deadline, err := optionalDate(item.Deadline1)
	if err != nil {
		return domain.Question{}, fmt.Errorf("converting the question %d: %w", item.ID, err)
	}

	responded, err := optionalDate(item.ResponseDate)
	if err != nil {
		return domain.Question{}, fmt.Errorf("converting the question %d: %w", item.ID, err)
	}

	questionType := ""
	if item.Type != nil {
		questionType = strings.TrimSpace(item.Type.Name)
	}

	return domain.Question{
		ID:                 item.ID,
		Type:               questionType,
		RegistrationNumber: strings.TrimSpace(item.RegistrationNumber),
		Registered:         registered,
		Request:            strings.TrimSpace(item.Request),
		Member:             text(item.ColleagueName),
		Faction:            text(item.NodeName),
		Addressee:          addressee(item),
		Deadline:           deadline,
		Responded:          responded,
		URL:                questionPage + strconv.Itoa(item.ID),
	}, nil
}

// addressee joins the body, the person, and the title that the question goes to.
func addressee(item parliamentclient.MPQuestion) string {
	targets := []*string{item.TargetOrganisation, item.TargetPerson, item.Target}
	parts := make([]string, 0, len(targets))

	for _, part := range targets {
		if value := text(part); value != "" {
			parts = append(parts, value)
		}
	}

	return strings.Join(parts, ", ")
}

// sourceRange writes the ends of a period in the form of the source. An absent end is empty.
func sourceRange(period domain.Period) (string, string) {
	return sourceText(period.From), sourceText(period.To)
}

func sourceText(date *kernel.Date) string {
	if date == nil {
		return ""
	}

	return date.Time().Format(sourceDate)
}

func sourceDateOf(value string) (kernel.Date, error) {
	parsed, err := time.Parse(sourceDate, strings.TrimSpace(value))
	if err != nil {
		return kernel.Date{}, fmt.Errorf("reading the date %q: %w", value, err)
	}

	date, err := kernel.ParseDate(parsed.Format(time.DateOnly))
	if err != nil {
		return kernel.Date{}, fmt.Errorf("reading the date %q: %w", value, err)
	}

	return date, nil
}

func optionalDate(value *string) (*kernel.Date, error) {
	if text(value) == "" {
		return nil, nil //nolint:nilnil // An absent date is not an error.
	}

	date, err := sourceDateOf(*value)
	if err != nil {
		return nil, err
	}

	return &date, nil
}

func text(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}

func unavailable(err error) error {
	return fmt.Errorf("reading the Parliament: %w: %w", kernel.ErrSourceUnavailable, err)
}
