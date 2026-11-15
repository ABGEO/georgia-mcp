package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	newYearKa = "ახალი წელი"
	newYearEn = "New Year's Day"
)

// HOLIDAY-SC-004: A holiday name trims both texts and falls back to the other language.
func TestHolidayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		georgian, english string
		wantKa, wantEn    string
		wantErr           error
	}{
		{
			name: "both", georgian: " ახალი წელი ", english: "New Year's Day ",
			wantKa: newYearKa, wantEn: newYearEn,
		},
		{
			name:     "no georgian",
			georgian: " ",
			english:  newYearEn,
			wantKa:   newYearEn,
			wantEn:   newYearEn,
		},
		{name: "no english", georgian: newYearKa, wantKa: newYearKa, wantEn: newYearKa},
		{name: "empty", georgian: "", english: " ", wantErr: domain.ErrInvalidHolidayName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			name, err := domain.NewHolidayName(test.georgian, test.english)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("NewHolidayName() error = %v, want %v", err, test.wantErr)
			}

			if err != nil {
				return
			}

			if got := name.In(kernel.LanguageGeorgian); got != test.wantKa {
				t.Errorf("In(ka) = %q, want %q", got, test.wantKa)
			}

			if got := name.In(kernel.LanguageEnglish); got != test.wantEn {
				t.Errorf("In(en) = %q, want %q", got, test.wantEn)
			}
		})
	}
}

// HOLIDAY-SC-004: A valid holiday name is never empty in a language.
func FuzzNewHolidayName(f *testing.F) {
	for _, seed := range [][2]string{{newYearKa, newYearEn}, {"", "x"}, {" ", ""}, {"\t", "\n"}} {
		f.Add(seed[0], seed[1])
	}

	f.Fuzz(func(t *testing.T, georgian, english string) {
		name, err := domain.NewHolidayName(georgian, english)
		if err != nil {
			return
		}

		for _, language := range []kernel.Language{kernel.LanguageGeorgian, kernel.LanguageEnglish} {
			if name.In(language) == "" {
				t.Errorf("NewHolidayName(%q, %q).In(%s) is empty", georgian, english, language)
			}
		}
	})
}

// HOLIDAY-SC-005: The weekday of a holiday is the weekday of its date.
func TestHolidayWeekday(t *testing.T) {
	t.Parallel()

	tests := []struct {
		date string
		want time.Weekday
	}{
		{date: "2026-05-26", want: time.Tuesday},
		{date: "2026-04-12", want: time.Sunday},
	}

	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			t.Parallel()

			date, err := kernel.ParseDate(test.date)
			if err != nil {
				t.Fatal(err)
			}

			if got := (domain.Holiday{Date: date}).Weekday(); got != test.want {
				t.Errorf("Weekday() = %s, want %s", got, test.want)
			}
		})
	}
}
