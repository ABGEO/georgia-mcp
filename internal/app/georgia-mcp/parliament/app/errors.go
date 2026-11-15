package app

import "errors"

var (
	// ErrBillNotFound tells that the Parliament has no bill with the identifier.
	ErrBillNotFound = errors.New("bill: not found")
	// ErrVoteNotFound tells that the Parliament has no vote with the identifier.
	ErrVoteNotFound = errors.New("vote: not found")
	// ErrNoCriterion tells that a search or a list has no text, no number, and no period.
	ErrNoCriterion = errors.New("search: no criterion")
)
