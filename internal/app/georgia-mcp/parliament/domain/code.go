package domain

import (
	"strings"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Reading is the code of a plenary review stage.
type Reading string

// The readings. OtherReading is a plenary stage with a name that the server does not know.
const (
	FirstReading        Reading = "first_reading"
	SecondReading       Reading = "second_reading"
	ThirdReading        Reading = "third_reading"
	SingleReading       Reading = "single_reading"
	PlenaryDraft        Reading = "plenary_draft"
	AlternativeProposal Reading = "alternative_proposal"
	OtherReading        Reading = "other"
)

// Outcome is the code of the result of a vote.
type Outcome string

// The outcomes.
const (
	Adopted      Outcome = "adopted"
	Rejected     Outcome = "rejected"
	OtherOutcome Outcome = "other"
)

// Choice is the code of what one member did in a vote.
type Choice string

// The choices. Absent is a member who did not vote.
const (
	Yes         Choice = "yes"
	No          Choice = "no"
	Absent      Choice = "absent"
	OtherChoice Choice = "other"
)

// QuestionStatus is the state of a member question.
type QuestionStatus string

// The question statuses.
const (
	Answered QuestionStatus = "answered"
	Pending  QuestionStatus = "pending"
	Overdue  QuestionStatus = "overdue"
)

// Coded is a code with the Georgian name that the Parliament gives it.
type Coded[C ~string] struct {
	Code C
	Name string
}

// ReadingOf returns the reading of a name of the Parliament, and false for any other name:
// a committee stage, or a plenary stage that the server does not know.
func ReadingOf(name string) (Reading, bool) {
	switch strings.TrimSpace(name) {
	case "I მოსმენა":
		return FirstReading, true
	case "II მოსმენა":
		return SecondReading, true
	case "III მოსმენა":
		return ThirdReading, true
	case "ერთი მოსმენა":
		return SingleReading, true
	case "სხდომაზე განსახილველი პროექტი":
		return PlenaryDraft, true
	case "ალტერნატიული წინადადება":
		return AlternativeProposal, true
	default:
		return "", false
	}
}

// OutcomeOf returns the outcome of a name of the Parliament, or OtherOutcome.
func OutcomeOf(name string) Outcome {
	switch strings.TrimSpace(name) {
	case "საკითხი მიღებულია":
		return Adopted
	case "საკითხი არ არის მიღებული":
		return Rejected
	default:
		return OtherOutcome
	}
}

// ChoiceOf returns the choice of a name of the Parliament, or OtherChoice.
func ChoiceOf(name string) Choice {
	switch strings.TrimSpace(name) {
	case "დიახ":
		return Yes
	case "არა":
		return No
	case "არ ესწრება":
		return Absent
	default:
		return OtherChoice
	}
}

// QuestionStatusOn returns the status of a question on the day. A question with a response
// is answered. A question with no response is overdue after its deadline, and pending
// before it, on it, or with no deadline.
func QuestionStatusOn(responded, deadline *kernel.Date, today kernel.Date) QuestionStatus {
	switch {
	case responded != nil:
		return Answered
	case deadline != nil && deadline.Before(today):
		return Overdue
	default:
		return Pending
	}
}
