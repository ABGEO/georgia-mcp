package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
)

func month(year int, m time.Month) domain.Month {
	return domain.Month{Year: year, Month: m}
}

func newest() domain.Month {
	return month(2026, time.September)
}

// figures marks a balance with the month or the year that it belongs to, in its total
// generation: 202603 for 2026-03, 2026 for the year 2026.
func figures(mark int64) domain.Figures {
	return domain.Figures{domain.GenerationTotal: decimal.NewFromInt(mark)}
}

type fakeBalance struct {
	calls int
}

func (f *fakeBalance) Balance(context.Context) (app.BalanceBook, error) {
	f.calls++

	var years []domain.YearBalance

	for year := 2022; year <= 2026; year++ {
		balance := domain.YearBalance{Year: year, Total: figures(int64(year))}

		last := time.December
		if year == 2026 {
			last = time.September
		}

		for m := time.January; m <= last; m++ {
			balance.Months = append(balance.Months, domain.MonthBalance{
				Month: month(year, m), Figures: figures(int64(year*100 + int(m))),
			})
		}

		years = append(years, balance)
	}

	return app.BalanceBook{Years: years}, nil
}

type fakePrices struct{}

func (fakePrices) Prices(context.Context) (app.PriceBook, error) {
	var months []domain.MonthPrices

	for m := month(2015, time.January); !newest().Before(m); m = m.AddMonths(1) {
		months = append(months, domain.MonthPrices{Month: m})
	}

	return app.PriceBook{Months: months}, nil
}

func getBalance(t *testing.T, query app.BalanceQuery) app.BalanceAnswer {
	t.Helper()

	answer, err := app.NewGetBalance(&fakeBalance{}).Get(t.Context(), query)
	if err != nil {
		t.Fatalf("Get(%+v) error = %v", query, err)
	}

	return answer
}

func checkBalance(t *testing.T, answer app.BalanceAnswer, period string, months int, mark int64) {
	t.Helper()

	total := answer.Figures[domain.GenerationTotal]
	if answer.Period != period || answer.Months != months ||
		!total.Equal(decimal.NewFromInt(mark)) ||
		answer.Newest != newest() {
		t.Errorf("answer = %s, %d months, %s, newest %s; want %s, %d, %d, 2026-09",
			answer.Period, answer.Months, total, answer.Newest, period, months, mark)
	}
}

// GNERCMON-SC-012: No month and no year gives the newest published month.
func TestGetBalanceGivesTheNewestMonth(t *testing.T) {
	t.Parallel()

	checkBalance(t, getBalance(t, app.BalanceQuery{}), "2026-09", 1, 202609)
}

// GNERCMON-SC-013: A month gives its figures. A year gives its total and the count of
// its published months.
func TestGetBalanceGivesAMonthOrAYear(t *testing.T) {
	t.Parallel()

	checkBalance(
		t,
		getBalance(t, app.BalanceQuery{Month: month(2026, time.March)}),
		"2026-03",
		1,
		202603,
	)
	checkBalance(t, getBalance(t, app.BalanceQuery{Year: 2026}), "2026", 9, 2026)
	checkBalance(t, getBalance(t, app.BalanceQuery{Year: 2025}), "2025", 12, 2025)
}

// GNERCMON-SC-014: A month with a year is refused before any call. A period outside the
// balance gives a *PeriodError with the accepted period.
func TestGetBalanceRefusesThePeriod(t *testing.T) {
	t.Parallel()

	source := &fakeBalance{}

	_, err := app.NewGetBalance(source).
		Get(t.Context(), app.BalanceQuery{Month: month(2026, time.March), Year: 2026})
	if !errors.Is(err, app.ErrMonthAndYear) || source.calls != 0 {
		t.Errorf("Get(month and year) error = %v after %d calls, want ErrMonthAndYear and none",
			err, source.calls)
	}

	tests := []struct {
		query app.BalanceQuery
		field string
		value string
	}{
		{app.BalanceQuery{Month: month(2021, time.December)}, "month", "2021-12"},
		{app.BalanceQuery{Month: month(2026, time.October)}, "month", "2026-10"},
		{app.BalanceQuery{Year: 2021}, "year", "2021"},
		{app.BalanceQuery{Year: 2027}, "year", "2027"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			_, err := app.NewGetBalance(&fakeBalance{}).Get(t.Context(), tt.query)
			checkPeriodError(t, err, tt.field, tt.value, month(2022, time.January))
		})
	}
}

func checkPeriodError(t *testing.T, err error, field, value string, first domain.Month) {
	t.Helper()

	var periodErr *app.PeriodError
	if !errors.As(err, &periodErr) || !errors.Is(err, app.ErrPeriodOutOfRange) ||
		periodErr.Field != field || periodErr.Value != value ||
		periodErr.First != first || periodErr.Newest != newest() {
		t.Errorf("error = %v (%+v), want a *PeriodError of %s %s from %s to 2026-09",
			err, periodErr, field, value, first)
	}
}

func listPrices(t *testing.T, query app.PriceQuery) app.PriceAnswer {
	t.Helper()

	answer, err := app.NewListPrices(fakePrices{}).List(t.Context(), query)
	if err != nil {
		t.Fatalf("List(%+v) error = %v", query, err)
	}

	return answer
}

func checkMonths(t *testing.T, answer app.PriceAnswer, first, last domain.Month, count int) {
	t.Helper()

	if len(answer.Months) != count || answer.Months[0].Month != first ||
		answer.Months[len(answer.Months)-1].Month != last || answer.Newest != newest() {
		t.Errorf("List() = %d months, newest %s; want %d from %s to %s, newest 2026-09",
			len(answer.Months), answer.Newest, count, first, last)
	}
}

// GNERCMON-SC-015: No period gives the last 12 published months.
func TestListPricesGivesTheLastYear(t *testing.T) {
	t.Parallel()

	checkMonths(t, listPrices(t, app.PriceQuery{}), month(2025, time.October), newest(), 12)
}

// GNERCMON-SC-016: A period ends at the newest published month. A missing end is the
// newest month. A missing start is 11 months before the end.
func TestListPricesGivesThePeriod(t *testing.T) {
	t.Parallel()

	checkMonths(t, listPrices(t, app.PriceQuery{
		From: month(2025, time.January), To: month(2025, time.December),
	}), month(2025, time.January), month(2025, time.December), 12)

	checkMonths(t, listPrices(t, app.PriceQuery{
		From: month(2026, time.January), To: month(2026, time.December),
	}), month(2026, time.January), newest(), 9)

	checkMonths(t, listPrices(t, app.PriceQuery{From: month(2026, time.January)}),
		month(2026, time.January), newest(), 9)

	checkMonths(t, listPrices(t, app.PriceQuery{To: month(2025, time.December)}),
		month(2025, time.January), month(2025, time.December), 12)

	checkMonths(t, listPrices(t, app.PriceQuery{To: month(2027, time.December)}),
		month(2025, time.October), newest(), 12)

	checkMonths(t, listPrices(t, app.PriceQuery{To: month(2015, time.June)}),
		month(2015, time.January), month(2015, time.June), 6)
}

// GNERCMON-SC-017: A start out of range, a reversed period, and a period of more than 60
// months are refused.
func TestListPricesRefusesThePeriod(t *testing.T) {
	t.Parallel()

	list := func(query app.PriceQuery) error {
		_, err := app.NewListPrices(fakePrices{}).List(t.Context(), query)

		return err
	}

	first := month(2015, time.January)
	checkPeriodError(
		t,
		list(app.PriceQuery{From: month(2014, time.December)}),
		"from",
		"2014-12",
		first,
	)
	checkPeriodError(
		t,
		list(app.PriceQuery{From: month(2026, time.October)}),
		"from",
		"2026-10",
		first,
	)

	checkPeriodError(
		t,
		list(app.PriceQuery{To: month(2014, time.December)}),
		"to",
		"2014-12",
		first,
	)

	err := list(app.PriceQuery{From: month(2025, time.June), To: month(2025, time.January)})
	if !errors.Is(err, app.ErrReversedPeriod) {
		t.Errorf("List(2025-06 to 2025-01) error = %v, want ErrReversedPeriod", err)
	}

	for _, tt := range []struct {
		query app.PriceQuery
		want  int
	}{
		{app.PriceQuery{From: month(2020, time.January), To: month(2025, time.December)}, 72},
		{app.PriceQuery{From: first}, 141},
	} {
		var spanErr *app.SpanError
		if err := list(
			tt.query,
		); !errors.As(err, &spanErr) ||
			!errors.Is(err, app.ErrPeriodTooLong) ||
			spanErr.Months != tt.want {
			t.Errorf("List(%+v) error = %v, want a *SpanError of %d", tt.query, err, tt.want)
		}
	}
}
