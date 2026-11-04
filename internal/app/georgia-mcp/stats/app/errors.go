package app

import (
	"errors"
	"fmt"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
)

var (
	// ErrInvalidQuery tells that the text of a search is empty or longer than 100 runes.
	ErrInvalidQuery = errors.New("query: invalid")
	// ErrCatalogLoading tells that the server has not read the catalog of the language yet.
	ErrCatalogLoading = errors.New("catalog: loading")
	// ErrUnknownTable tells that Geostat has no table with the code.
	ErrUnknownTable = errors.New("table: unknown")
	// ErrInvalidSelection tells that a selection names a dimension or a value that the
	// table does not have, or selects a dimension in a way that is not valid.
	ErrInvalidSelection = errors.New("selection: invalid")
	// ErrTooManyCells tells that a selection holds more cells than a request can have.
	ErrTooManyCells = errors.New("selection: too many cells")
	// ErrMonthNotPublished tells that Geostat published no consumer price index for the
	// month.
	ErrMonthNotPublished = errors.New("month: not published")
)

// SelectionProblem is the reason that a selection is refused.
type SelectionProblem int

// The problems of a selection.
const (
	ProblemUnknownDimension SelectionProblem = iota + 1
	ProblemUnknownValue
	ProblemValuesOrLast
	ProblemDuplicate
	ProblemTooManyCells
)

// SelectionError tells why a selection is refused. Accepted holds the codes that the
// dimension or the table accepts. It wraps ErrTooManyCells for ProblemTooManyCells, and
// ErrInvalidSelection for every other problem.
type SelectionError struct {
	Problem   SelectionProblem
	Dimension string
	Value     string
	Accepted  []string
	Cells     int
	err       error
}

func newSelectionError(problem SelectionProblem, dimension string) *SelectionError {
	sentinel := ErrInvalidSelection
	if problem == ProblemTooManyCells {
		sentinel = ErrTooManyCells
	}

	return &SelectionError{Problem: problem, Dimension: dimension, err: sentinel}
}

// Error names the problem, the dimension, and the value.
func (e *SelectionError) Error() string {
	return fmt.Sprintf("%s: dimension %q, value %q, %d cells", e.err, e.Dimension, e.Value, e.Cells)
}

// Unwrap returns the sentinel of the problem.
func (e *SelectionError) Unwrap() error {
	return e.err
}

// MonthError tells that Geostat published no index for the month, and names the newest
// month with an index. Newest is the zero month when Geostat published none.
type MonthError struct {
	Month  domain.Month
	Newest domain.Month
}

// Error names the month and the newest month.
func (e *MonthError) Error() string {
	return fmt.Sprintf("%s: %s, the newest is %s", ErrMonthNotPublished, e.Month, e.Newest)
}

// Unwrap returns ErrMonthNotPublished.
func (e *MonthError) Unwrap() error {
	return ErrMonthNotPublished
}
