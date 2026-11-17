package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
)

func month(year int, m time.Month) domain.Month {
	return domain.Month{Year: year, Month: m}
}

// GNERCMON-SC-006: ParseMonth takes YYYY-MM only. String writes it back.
func TestParseMonth(t *testing.T) {
	t.Parallel()

	got, err := domain.ParseMonth("2026-03")
	if err != nil || got != month(2026, time.March) || got.String() != "2026-03" {
		t.Errorf("ParseMonth(2026-03) = %v, %v, want March 2026", got, err)
	}

	for _, text := range []string{"2026-3", "2026-13", "March", "2026-03-01", "", "2026-00", "0000-01"} {
		if _, err := domain.ParseMonth(text); !errors.Is(err, domain.ErrInvalidMonth) {
			t.Errorf("ParseMonth(%q) error = %v, want ErrInvalidMonth", text, err)
		}
	}
}

func FuzzParseMonth(f *testing.F) {
	for _, seed := range []string{"2026-03", "2015-01", "2026-3", "March", ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got, err := domain.ParseMonth(input)
		if err != nil {
			return
		}

		back, err := domain.ParseMonth(got.String())
		if err != nil || back != got {
			t.Errorf("ParseMonth(%q) = %v, which does not parse back: %v", input, got, err)
		}
	})
}

// GNERCMON-SC-007: AddMonths crosses a year. MonthsFrom counts both ends.
func TestMonthArithmetic(t *testing.T) {
	t.Parallel()

	january := month(2026, time.January)

	if got := january.AddMonths(-1); got != month(2025, time.December) {
		t.Errorf("AddMonths(-1) = %v, want 2025-12", got)
	}

	if got := january.AddMonths(11); got != month(2026, time.December) {
		t.Errorf("AddMonths(11) = %v, want 2026-12", got)
	}

	if got := month(2025, time.December).MonthsFrom(month(2025, time.January)); got != 12 {
		t.Errorf("2025-01 to 2025-12 = %d months, want 12", got)
	}

	if got := month(2025, time.December).MonthsFrom(month(2020, time.January)); got != 72 {
		t.Errorf("2020-01 to 2025-12 = %d months, want 72", got)
	}
}

// GNERCMON-SC-007: Before orders by year, then by month. IsZero holds for the zero month.
func TestMonthOrder(t *testing.T) {
	t.Parallel()

	january := month(2026, time.January)

	if !month(2025, time.December).Before(january) || january.Before(month(2025, time.December)) ||
		!month(2026, time.January).Before(month(2026, time.February)) || january.Before(january) {
		t.Error("Before does not order by year, then by month")
	}

	if !(domain.Month{}).IsZero() || january.IsZero() {
		t.Error("IsZero holds for the zero month only")
	}
}

// GNERCMON-SC-008: Round keeps 3 decimals and rounds half away from zero.
func TestRound(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"388.95957600000003": "388.960",
		"1.0005":             "1.001",
		"-1.0005":            "-1.001",
		"0":                  "0.000",
	} {
		got := domain.Round(decimal.RequireFromString(input)).StringFixed(domain.Precision)
		if got != want {
			t.Errorf("Round(%s) = %s, want %s", input, got, want)
		}
	}
}

// GNERCMON-SC-009: Items lists the 34 items of the balance, each once, and the five
// required ones.
func TestItems(t *testing.T) {
	t.Parallel()

	items := domain.Items()
	seen := map[domain.Item]bool{}
	required := 0

	for _, item := range items {
		if seen[item] {
			t.Errorf("Items() holds %s twice", item)
		}

		seen[item] = true

		if item.Required() {
			required++
		}
	}

	if len(items) != 34 || required != 5 {
		t.Errorf("Items() = %d items, %d required, want 34 and 5", len(items), required)
	}
}
