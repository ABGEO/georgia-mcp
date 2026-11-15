package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
)

// BillFilter selects the bills by a text of the title, a registration number, and a
// period of registration. The zero value of each is absent.
type BillFilter struct {
	Text       domain.SearchText
	Number     string
	Registered domain.Period
}

// IsZero reports whether the filter has no criterion.
func (f BillFilter) IsZero() bool {
	return f.Text.IsZero() && f.Number == "" && f.Registered.IsZero()
}

// VoteFilter selects the votes by a text of the title of the bill and a period of the
// vote. The zero value of each is absent.
type VoteFilter struct {
	BillTitle domain.SearchText
	Voted     domain.Period
}

// IsZero reports whether the filter has no criterion.
func (f VoteFilter) IsZero() bool {
	return f.BillTitle.IsZero() && f.Voted.IsZero()
}

// QuestionFilter selects the member questions by a text of the request, a name of the
// member, and a period of registration. The zero value of each is absent.
type QuestionFilter struct {
	Text       domain.SearchText
	Member     domain.SearchText
	Registered domain.Period
}

// IsZero reports whether the filter has no criterion.
func (f QuestionFilter) IsZero() bool {
	return f.Text.IsZero() && f.Member.IsZero() && f.Registered.IsZero()
}

// BillPage is one page of bills, the count of every match, and the instant of the read.
type BillPage struct {
	Bills  []domain.Bill
	Total  int
	ReadAt time.Time
	Stale  bool
}

// BillRead is one bill with its stages in the order of the review.
type BillRead struct {
	Bill   domain.Bill
	Stages []domain.Stage
	ReadAt time.Time
	Stale  bool
}

// VotePage is one page of votes, the count of every match, and the instant of the read.
type VotePage struct {
	Votes  []domain.Vote
	Total  int
	ReadAt time.Time
	Stale  bool
}

// VoteRead is one vote with the choice of each member, in the order of the source.
type VoteRead struct {
	Vote    domain.Vote
	Choices []domain.MemberChoice
	ReadAt  time.Time
	Stale   bool
}

// QuestionPage is one page of member questions, the count of every match, and the instant
// of the read.
type QuestionPage struct {
	Questions []domain.Question
	Total     int
	ReadAt    time.Time
	Stale     bool
}

// BillSource gives the bills of the Parliament.
type BillSource interface {
	// SearchBills reads one page of the bills that match the filter, newest first.
	SearchBills(ctx context.Context, filter BillFilter, page domain.Page) (BillPage, error)
	// Bill reads the bill, its stages, and the votes of each stage, or ErrBillNotFound.
	Bill(ctx context.Context, id domain.ID) (BillRead, error)
}

// VoteSource gives the plenary votes of the Parliament.
type VoteSource interface {
	// ListVotes reads one page of the votes that match the filter, newest first.
	ListVotes(ctx context.Context, filter VoteFilter, page domain.Page) (VotePage, error)
	// Vote reads the vote and the choice of each member, or ErrVoteNotFound.
	Vote(ctx context.Context, id domain.ID) (VoteRead, error)
}

// QuestionSource gives the member questions of the Parliament.
type QuestionSource interface {
	// SearchQuestions reads one page of the questions that match the filter, newest first.
	SearchQuestions(
		ctx context.Context,
		filter QuestionFilter,
		page domain.Page,
	) (QuestionPage, error)
}
