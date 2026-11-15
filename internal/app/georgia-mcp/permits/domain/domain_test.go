package domain_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	approvedNo = "AR11168188"
	oneDay     = "2026-11-04"
)

func date(t *testing.T, text string) kernel.Date {
	t.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func number(t *testing.T, text string) domain.ApplicationNumber {
	t.Helper()

	parsed, err := domain.ParseApplicationNumber(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// TASPERM-SC-007: A code of 4 or 5 groups parses. Any other text fails, a code of a unit
// too.
func TestParseCadastralCode(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"01.18.04.020.005", "53.20.41.012", " 05.31.07.1000 "} {
		if code, err := domain.ParseCadastralCode(text); err != nil || code.IsZero() {
			t.Errorf("ParseCadastralCode(%q) = %v, %v, want a code", text, code, err)
		}
	}

	for _, text := range []string{"01.18", "", "01.14.11.005.029.01.501", "01.18.04.020.00a"} {
		if _, err := domain.ParseCadastralCode(
			text,
		); !errors.Is(
			err,
			domain.ErrInvalidCadastralCode,
		) {
			t.Errorf("ParseCadastralCode(%q) error = %v, want ErrInvalidCadastralCode", text, err)
		}
	}
}

// TASPERM-SC-007: A valid code gives itself from String.
func FuzzParseCadastralCode(f *testing.F) {
	for _, seed := range []string{"01.18.04.020.005", "53.20.41.012", "", "1.2.3.4.5.6"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		code, err := domain.ParseCadastralCode(text)
		if err != nil {
			return
		}

		again, err := domain.ParseCadastralCode(code.String())
		if err != nil || again != code {
			t.Errorf("ParseCadastralCode(%q) = %q, and again %q, %v", text, code, again, err)
		}
	})
}

// TASPERM-SC-007: A number with AR1, in any case, or digits alone parse.
func TestParseApplicationNumber(t *testing.T) {
	t.Parallel()

	for _, text := range []string{approvedNo, " ar11168188 ", "1168188"} {
		if got, err := domain.ParseApplicationNumber(
			text,
		); err != nil ||
			got.String() != approvedNo {
			t.Errorf("ParseApplicationNumber(%q) = %q, %v, want %s", text, got, err, approvedNo)
		}
	}

	for _, text := range []string{"AR1116818X", "AR2123", "", "AR1", "1234567890"} {
		_, err := domain.ParseApplicationNumber(text)
		if !errors.Is(err, domain.ErrInvalidApplicationNumber) {
			t.Errorf(
				"ParseApplicationNumber(%q) error = %v, want ErrInvalidApplicationNumber",
				text,
				err,
			)
		}
	}
}

// TASPERM-SC-007: A valid number gives itself from String.
func FuzzParseApplicationNumber(f *testing.F) {
	for _, seed := range []string{approvedNo, "1168188", "ar1", ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		parsed, err := domain.ParseApplicationNumber(text)
		if err != nil {
			return
		}

		again, err := domain.ParseApplicationNumber(parsed.String())
		if err != nil || again != parsed {
			t.Errorf("ParseApplicationNumber(%q) = %q, and again %q, %v", text, parsed, again, err)
		}
	})
}

// TASPERM-SC-008: A period holds at most 31 days, both ends included, and starts no later
// than it ends.
func TestNewPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from, to string
		want     error
	}{
		{"2026-11-01", "2026-12-01", nil},
		{oneDay, oneDay, nil},
		{"2026-11-01", "2026-12-02", domain.ErrPeriodTooLong},
		{"2026-11-10", oneDay, domain.ErrInvalidPeriod},
	}

	for _, tt := range tests {
		period, err := domain.NewPeriod(date(t, tt.from), date(t, tt.to))
		if !errors.Is(err, tt.want) {
			t.Errorf("NewPeriod(%s, %s) error = %v, want %v", tt.from, tt.to, err, tt.want)
		}

		if err == nil && (period.IsZero() || period.From.String() != tt.from) {
			t.Errorf("NewPeriod(%s, %s) = %v", tt.from, tt.to, period)
		}
	}
}

// TASPERM-SC-008: A period never panics, and a valid one keeps its bounds.
func FuzzNewPeriod(f *testing.F) {
	f.Add(0, 30)
	f.Add(10, 0)
	f.Add(-5, 400)

	f.Fuzz(func(t *testing.T, from, to int) {
		start := date(t, "2026-01-01")

		period, err := domain.NewPeriod(start.AddDays(from%5000), start.AddDays(to%5000))
		if err != nil {
			return
		}

		days := kernel.DaysBetween(period.From, period.To) + 1
		if days < 1 || days > domain.MaxPeriodDays {
			t.Errorf("NewPeriod() = %v, %d days", period, days)
		}
	})
}

// TASPERM-SC-012: The newest registration comes first, and on one day the highest number.
func TestSortApplications(t *testing.T) {
	t.Parallel()

	day, before := date(t, oneDay), date(t, "2026-11-03")
	applications := []domain.Application{
		{Number: number(t, "AR1506582"), Registered: day},
		{Number: number(t, "AR11168000"), Registered: before},
		{Number: number(t, approvedNo), Registered: day},
	}

	sorted := domain.SortApplications(applications)

	got := make([]string, 0, len(sorted))
	for _, application := range sorted {
		got = append(got, application.Number.String())
	}

	want := []string{approvedNo, "AR1506582", "AR11168000"}
	if !slices.Equal(got, want) {
		t.Errorf("SortApplications() = %v, want %v", got, want)
	}

	if applications[0].Number.String() != "AR1506582" {
		t.Error("SortApplications() changed its argument")
	}
}
