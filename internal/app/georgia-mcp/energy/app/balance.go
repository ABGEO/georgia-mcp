package app

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// BalanceQuery asks for the balance of a month, of a year, or of the newest month.
type BalanceQuery struct {
	// Month is zero for no month.
	Month domain.Month
	// Year is 0 for no year.
	Year int
}

// BalanceAnswer is the balance of one month or one year.
type BalanceAnswer struct {
	// Period is "2026-03" for a month, "2026" for a year.
	Period string
	// Months is the count of published months that the period holds.
	Months   int
	Figures  domain.Figures
	Newest   domain.Month
	Modified time.Time
	ReadAt   time.Time
	Stale    bool
}

// GetBalance gives the balance of a month or a year.
type GetBalance struct {
	source BalanceSource
}

// NewGetBalance returns the use case over source.
func NewGetBalance(source BalanceSource) *GetBalance {
	return &GetBalance{source: source}
}

// Get returns the balance that query names. It refuses a month with a year with
// ErrMonthAndYear, and a period outside the balance with a *PeriodError.
func (g *GetBalance) Get(ctx context.Context, query BalanceQuery) (BalanceAnswer, error) {
	if !query.Month.IsZero() && query.Year != 0 {
		return BalanceAnswer{}, ErrMonthAndYear
	}

	book, err := g.source.Balance(ctx)
	if err != nil {
		return BalanceAnswer{}, fmt.Errorf("reading the balance: %w", err)
	}

	if len(book.Years) == 0 || len(book.Years[len(book.Years)-1].Months) == 0 {
		return BalanceAnswer{}, fmt.Errorf(
			"reading the balance: %w: %w",
			errNoData,
			kernel.ErrSourceUnavailable,
		)
	}

	lastYear := book.Years[len(book.Years)-1]
	newest := lastYear.Months[len(lastYear.Months)-1].Month
	answer := BalanceAnswer{
		Newest:   newest,
		Modified: book.Modified,
		ReadAt:   book.ReadAt,
		Stale:    book.Stale,
	}

	if query.Year != 0 {
		year, ok := findYear(book.Years, query.Year)
		if !ok {
			return BalanceAnswer{}, balanceError("year", strconv.Itoa(query.Year), newest)
		}

		answer.Period, answer.Months, answer.Figures = strconv.Itoa(
			year.Year,
		), len(
			year.Months,
		), year.Total

		return answer, nil
	}

	target := query.Month
	if target.IsZero() {
		target = newest
	}

	figures, ok := findMonth(book.Years, target)
	if !ok {
		return BalanceAnswer{}, balanceError("month", target.String(), newest)
	}

	answer.Period, answer.Months, answer.Figures = target.String(), 1, figures

	return answer, nil
}

func balanceError(field, value string, newest domain.Month) *PeriodError {
	return &PeriodError{
		Field: field, Value: value,
		First:  domain.Month{Year: domain.FirstBalanceYear, Month: time.January},
		Newest: newest,
	}
}

func findYear(years []domain.YearBalance, number int) (domain.YearBalance, bool) {
	for _, year := range years {
		if year.Year == number && number >= domain.FirstBalanceYear {
			return year, true
		}
	}

	return domain.YearBalance{}, false
}

func findMonth(years []domain.YearBalance, target domain.Month) (domain.Figures, bool) {
	year, ok := findYear(years, target.Year)
	if !ok {
		return nil, false
	}

	for _, month := range year.Months {
		if month.Month == target {
			return month.Figures, true
		}
	}

	return nil, false
}
