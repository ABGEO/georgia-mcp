package domain

import (
	"cmp"
	"slices"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// MaxMatches is the most applications that a search reads and sorts.
const MaxMatches = 500

// Status is the state of an application.
type Status string

// The statuses of an application.
const (
	StatusApproved        Status = "approved"
	StatusRefused         Status = "refused"
	StatusDeficient       Status = "deficient"
	StatusConditional     Status = "conditional"
	StatusNotReviewed     Status = "not_reviewed"
	StatusRevoked         Status = "revoked"
	StatusCompleted       Status = "completed"
	StatusAwaitingPayment Status = "awaiting_payment"
	StatusInReview        Status = "in_review"
)

// Application is one request to TAS. Every text is the text of TAS.
type Application struct {
	Number         ApplicationNumber
	URL            string
	Address        string
	Registered     kernel.Date
	Deadline       *kernel.Date
	Status         Status
	Category       string
	ParentCategory string
	Class          string
	Stages         []string
	Action         string
	Executor       string
	// DecisionText is the decision as plain text. It is empty in a search.
	DecisionText string
}

// SortApplications orders by the date of registration, newest first, then by number,
// highest first. It returns a new slice.
func SortApplications(applications []Application) []Application {
	sorted := slices.Clone(applications)

	slices.SortStableFunc(sorted, func(first, second Application) int {
		switch {
		case first.Registered.After(second.Registered):
			return -1
		case first.Registered.Before(second.Registered):
			return 1
		default:
			return cmp.Compare(second.Number.digits(), first.Number.digits())
		}
	})

	return sorted
}
