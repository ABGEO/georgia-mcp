package app

import (
	"context"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// QuestionItem is one member question with its status on the current day.
type QuestionItem struct {
	Question domain.Question
	Status   domain.QuestionStatus
}

// QuestionResult is one page of member questions, with the notice of a text that is not
// Georgian.
type QuestionResult struct {
	Questions []QuestionItem
	Total     int
	ReadAt    time.Time
	Stale     bool
	Notice    string
}

// SearchQuestions returns the member questions that match a filter, newest first.
type SearchQuestions struct {
	source QuestionSource
	now    func() time.Time
}

// NewSearchQuestions returns the use case over the source. now gives the current day.
func NewSearchQuestions(source QuestionSource, now func() time.Time) *SearchQuestions {
	return &SearchQuestions{source: source, now: now}
}

// Search refuses a filter with no criterion, reads one page, and sets the status of each
// question on the current day in Tbilisi.
func (u *SearchQuestions) Search(
	ctx context.Context,
	filter QuestionFilter,
	page domain.Page,
) (QuestionResult, error) {
	if filter.IsZero() {
		return QuestionResult{}, ErrNoCriterion
	}

	read, err := u.source.SearchQuestions(ctx, filter, page)
	if err != nil {
		return QuestionResult{}, fmt.Errorf("searching the member questions: %w", err)
	}

	today := kernel.Today(u.now())
	items := make([]QuestionItem, 0, len(read.Questions))

	for _, question := range read.Questions {
		items = append(items, QuestionItem{
			Question: question,
			Status:   domain.QuestionStatusOn(question.Responded, question.Deadline, today),
		})
	}

	return QuestionResult{
		Questions: items,
		Total:     read.Total,
		ReadAt:    read.ReadAt,
		Stale:     read.Stale,
		Notice:    notice(read.Total, filter.Text, filter.Member),
	}, nil
}
