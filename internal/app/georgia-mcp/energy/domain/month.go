package domain

import (
	"cmp"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

const (
	// FirstBalanceYear is the first year of the balance of GNERC that the context reads.
	FirstBalanceYear = 2022
	// FirstPriceYear is the first year of the prices of GNERC.
	FirstPriceYear = 2015
	// MaxPriceMonths bounds a period of prices.
	MaxPriceMonths = 60
	// DefaultPriceMonths is the length of a period of prices with no start.
	DefaultPriceMonths = 12
	// Precision is the count of decimals of each value of GNERC.
	Precision = 3

	monthLayout     = "2006-01"
	monthsInYear    = 12
	monthTextLength = len(monthLayout)
)

// Month is a month of the calendar. The zero Month is no month.
type Month struct {
	Year  int
	Month time.Month
}

// ParseMonth reads "2026-03". Any other text gives ErrInvalidMonth.
func ParseMonth(text string) (Month, error) {
	parsed, err := time.Parse(monthLayout, text)
	if err != nil || len(text) != monthTextLength || parsed.Year() < 1 {
		return Month{}, fmt.Errorf("%w: %q", ErrInvalidMonth, text)
	}

	return Month{Year: parsed.Year(), Month: parsed.Month()}, nil
}

// String writes the month as YYYY-MM.
func (m Month) String() string {
	return fmt.Sprintf("%04d-%02d", m.Year, int(m.Month))
}

// AddMonths returns the month n months after m. A negative n goes back.
func (m Month) AddMonths(n int) Month {
	index := m.index() + n

	return Month{Year: index / monthsInYear, Month: time.Month(index%monthsInYear + 1)}
}

// Before tells whether m comes before o.
func (m Month) Before(o Month) bool {
	return m.index() < o.index()
}

// Compare returns -1 when m comes before o, 0 when they are the same, and +1 after.
func (m Month) Compare(o Month) int {
	return cmp.Compare(m.index(), o.index())
}

// IsZero tells whether m is no month.
func (m Month) IsZero() bool {
	return m == Month{}
}

// MonthsFrom returns the count of months from start to m, both included: 2025-01 to
// 2025-12 is 12.
func (m Month) MonthsFrom(start Month) int {
	return m.index() - start.index() + 1
}

func (m Month) index() int {
	return m.Year*monthsInYear + int(m.Month) - 1
}

// Round rounds half away from zero to Precision decimals.
func Round(d decimal.Decimal) decimal.Decimal {
	return d.Round(Precision)
}
