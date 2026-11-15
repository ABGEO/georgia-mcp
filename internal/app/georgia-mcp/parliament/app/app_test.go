package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

var errOutage = errors.New("outage")

// fakeSource implements the three ports. It answers with its totals, counts the reads,
// and fails with err when it is set.
type fakeSource struct {
	total     int
	questions []domain.Question
	choices   []domain.MemberChoice
	err       error
	reads     int
}

func (f *fakeSource) SearchBills(
	context.Context,
	app.BillFilter,
	domain.Page,
) (app.BillPage, error) {
	f.reads++

	return app.BillPage{Total: f.total, ReadAt: clock()}, f.err
}

func (f *fakeSource) Bill(_ context.Context, id domain.ID) (app.BillRead, error) {
	f.reads++

	return app.BillRead{Bill: domain.Bill{ID: id}, ReadAt: clock()}, f.err
}

func (f *fakeSource) ListVotes(context.Context, app.VoteFilter, domain.Page) (app.VotePage, error) {
	f.reads++

	return app.VotePage{Total: f.total, ReadAt: clock()}, f.err
}

func (f *fakeSource) Vote(_ context.Context, id domain.ID) (app.VoteRead, error) {
	f.reads++

	return app.VoteRead{Vote: domain.Vote{ID: id}, Choices: f.choices, ReadAt: clock()}, f.err
}

func (f *fakeSource) SearchQuestions(
	context.Context,
	app.QuestionFilter,
	domain.Page,
) (app.QuestionPage, error) {
	f.reads++

	return app.QuestionPage{Questions: f.questions, Total: f.total, ReadAt: clock()}, f.err
}

var (
	_ app.BillSource     = (*fakeSource)(nil)
	_ app.VoteSource     = (*fakeSource)(nil)
	_ app.QuestionSource = (*fakeSource)(nil)
)

func text(t *testing.T, value string) domain.SearchText {
	t.Helper()

	parsed, err := domain.NewSearchText(value)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func date(t *testing.T, value string) *kernel.Date {
	t.Helper()

	parsed, err := kernel.ParseDate(value)
	if err != nil {
		t.Fatal(err)
	}

	return &parsed
}

func firstPage() domain.Page {
	return domain.Page{Number: 1, Size: 10}
}

// PARL-SC-011: Each search with a zero filter fails with ErrNoCriterion and reads nothing.
func TestSearchesNeedACriterion(t *testing.T) {
	t.Parallel()

	source := &fakeSource{}

	_, billErr := app.NewSearchBills(source).Search(t.Context(), app.BillFilter{}, firstPage())
	_, voteErr := app.NewListVotes(source).List(t.Context(), app.VoteFilter{}, firstPage())
	_, questionErr := app.NewSearchQuestions(source, clock).
		Search(t.Context(), app.QuestionFilter{}, firstPage())

	for _, err := range []error{billErr, voteErr, questionErr} {
		if !errors.Is(err, app.ErrNoCriterion) {
			t.Errorf("error = %v, want ErrNoCriterion", err)
		}
	}

	if source.reads != 0 {
		t.Errorf("reads = %d, want 0", source.reads)
	}
}

// PARL-SC-011: Each criterion alone is enough.
func TestEachCriterionIsEnough(t *testing.T) {
	t.Parallel()

	period, err := domain.NewPeriod(date(t, "2026-11-10"), nil)
	if err != nil {
		t.Fatal(err)
	}

	source := &fakeSource{total: 1}
	bills := app.NewSearchBills(source)
	votes := app.NewListVotes(source)
	questions := app.NewSearchQuestions(source, clock)

	calls := map[string]error{}
	_, calls["bill text"] = bills.Search(
		t.Context(),
		app.BillFilter{Text: text(t, "ბიუჯეტ")},
		firstPage(),
	)
	_, calls["bill number"] = bills.Search(
		t.Context(),
		app.BillFilter{Number: "372/2-XIმპ"},
		firstPage(),
	)
	_, calls["bill period"] = bills.Search(
		t.Context(),
		app.BillFilter{Registered: period},
		firstPage(),
	)
	_, calls["vote title"] = votes.List(
		t.Context(),
		app.VoteFilter{BillTitle: text(t, "ბიუჯეტ")},
		firstPage(),
	)
	_, calls["vote period"] = votes.List(t.Context(), app.VoteFilter{Voted: period}, firstPage())
	_, calls["question text"] = questions.Search(
		t.Context(),
		app.QuestionFilter{Text: text(t, "ბიუჯეტ")},
		firstPage(),
	)
	_, calls["question member"] = questions.Search(
		t.Context(),
		app.QuestionFilter{Member: text(t, "ხორგუანი")},
		firstPage(),
	)
	_, calls["question period"] = questions.Search(
		t.Context(),
		app.QuestionFilter{Registered: period},
		firstPage(),
	)

	for name, err := range calls {
		if err != nil {
			t.Errorf("%s: error = %v", name, err)
		}
	}

	if source.reads != len(calls) {
		t.Errorf("reads = %d, want %d", source.reads, len(calls))
	}
}

// PARL-SC-022: A text with no Georgian letter that finds nothing gives the notice. A
// Georgian text, or a text that finds something, gives none.
func TestNotice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		total int
		want  string
	}{
		{name: "English, nothing", text: "budget", want: app.GeorgianOnlyNotice},
		{name: "English, a match", text: "budget", total: 3},
		{name: "Georgian, nothing", text: "zzqxვ"},
	}

	for _, tt := range tests {
		source := &fakeSource{total: tt.total}

		bills, err := app.NewSearchBills(source).
			Search(t.Context(), app.BillFilter{Text: text(t, tt.text)}, firstPage())
		if err != nil || bills.Notice != tt.want {
			t.Errorf("%s: bills notice = %q, %v, want %q", tt.name, bills.Notice, err, tt.want)
		}

		votes, err := app.NewListVotes(source).
			List(t.Context(), app.VoteFilter{BillTitle: text(t, tt.text)}, firstPage())
		if err != nil || votes.Notice != tt.want {
			t.Errorf("%s: votes notice = %q, %v, want %q", tt.name, votes.Notice, err, tt.want)
		}

		questions, err := app.NewSearchQuestions(source, clock).
			Search(t.Context(), app.QuestionFilter{Member: text(t, tt.text)}, firstPage())
		if err != nil || questions.Notice != tt.want {
			t.Errorf(
				"%s: questions notice = %q, %v, want %q",
				tt.name,
				questions.Notice,
				err,
				tt.want,
			)
		}
	}
}

// PARL-SC-022: A period with no vote gives no notice.
func TestNoNoticeWithNoText(t *testing.T) {
	t.Parallel()

	period, err := domain.NewPeriod(date(t, "2026-11-10"), nil)
	if err != nil {
		t.Fatal(err)
	}

	votes, err := app.NewListVotes(&fakeSource{}).
		List(t.Context(), app.VoteFilter{Voted: period}, firstPage())
	if err != nil || votes.Notice != "" {
		t.Errorf("notice = %q, %v, want none", votes.Notice, err)
	}
}

// PARL-SC-009: The search sets the status of each question on the day of the clock.
func TestSearchQuestionsSetsTheStatus(t *testing.T) {
	t.Parallel()

	source := &fakeSource{
		total: 3,
		questions: []domain.Question{
			{ID: 1, Deadline: date(t, "2026-11-28")},
			{ID: 2, Deadline: date(t, "2026-11-14")},
			{ID: 3, Deadline: date(t, "2026-10-01"), Responded: date(t, "2026-09-30")},
		},
	}

	result, err := app.NewSearchQuestions(source, clock).
		Search(t.Context(), app.QuestionFilter{Text: text(t, "ბიუჯეტ")}, firstPage())
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	got := make([]domain.QuestionStatus, 0, len(result.Questions))
	for _, item := range result.Questions {
		got = append(got, item.Status)
	}

	want := []domain.QuestionStatus{domain.Pending, domain.Overdue, domain.Answered}
	if !reflect.DeepEqual(got, want) || result.Total != 3 || !result.ReadAt.Equal(clock()) {
		t.Errorf("statuses = %v, total %d, want %v, total 3", got, result.Total, want)
	}
}

// PARL-SC-010: A vote holds the tally of its choices by faction.
func TestGetVoteTallies(t *testing.T) {
	t.Parallel()

	yes := domain.Coded[domain.Choice]{Code: domain.Yes, Name: "დიახ"}
	source := &fakeSource{choices: []domain.MemberChoice{
		{Member: "a", Faction: "ა", Choice: yes},
		{Member: "b", Choice: yes},
	}}

	detail, err := app.NewGetVote(source).Get(t.Context(), 31507)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if detail.Vote.ID != 31507 || len(detail.Factions) != 2 || detail.Factions[0].Faction != "ა" ||
		detail.Factions[0].Yes != 1 || detail.Factions[1].Faction != "" {
		t.Errorf("Get() = %+v", detail)
	}
}

// PARL-SC-008: A failure of the source passes through each use case, with its sentinel.
func TestFailuresPassThrough(t *testing.T) {
	t.Parallel()

	for _, sentinel := range []error{errOutage, app.ErrBillNotFound, app.ErrVoteNotFound} {
		source := &fakeSource{err: sentinel}
		filter := app.BillFilter{Text: text(t, "ბიუჯეტ")}

		_, searchErr := app.NewSearchBills(source).Search(t.Context(), filter, firstPage())
		_, billErr := app.NewGetBill(source).Get(t.Context(), 32449)
		_, votesErr := app.NewListVotes(source).
			List(t.Context(), app.VoteFilter{BillTitle: filter.Text}, firstPage())
		_, voteErr := app.NewGetVote(source).Get(t.Context(), 31507)
		_, questionErr := app.NewSearchQuestions(source, clock).
			Search(t.Context(), app.QuestionFilter{Text: filter.Text}, firstPage())

		for _, err := range []error{searchErr, billErr, votesErr, voteErr, questionErr} {
			if !errors.Is(err, sentinel) {
				t.Errorf("error = %v, want %v", err, sentinel)
			}
		}
	}
}
