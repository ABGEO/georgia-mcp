package kernel_test

import (
	"errors"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// NBGRATE-SC-009: A real calendar date in YYYY-MM-DD is a date. Any other text is not.
func TestParseDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		wantErr error
	}{
		{input: "2026-10-17"},
		{input: "2026-02-30", wantErr: kernel.ErrInvalidDate},
		{input: "2026-13-45", wantErr: kernel.ErrInvalidDate},
		{input: "17/10/2026", wantErr: kernel.ErrInvalidDate},
		{input: "2026-1-05", wantErr: kernel.ErrInvalidDate},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseDate(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseDate(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if tt.wantErr == nil && got.String() != tt.input {
				t.Errorf("ParseDate(%q) = %q", tt.input, got)
			}
		})
	}
}

// NBGRATE-SC-009: No input panics, and every accepted date formats back to its input.
func FuzzParseDate(f *testing.F) {
	for _, seed := range []string{"2026-10-17", "2026-02-30", "0000-00-00", "9999-12-31", "2026-1-5"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		date, err := kernel.ParseDate(input)
		if err != nil {
			return
		}

		if date.String() != input {
			t.Fatalf("ParseDate(%q).String() = %q", input, date)
		}
	})
}

func TestToday(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 17, 21, 30, 0, 0, time.UTC)

	if got := kernel.Today(now).String(); got != "2026-10-18" {
		t.Errorf("Today(%v) = %s, want 2026-10-18", now, got)
	}
}

func TestDateArithmetic(t *testing.T) {
	t.Parallel()

	from := mustDate(t, "2026-09-17")
	to := mustDate(t, "2026-10-17")

	if got := kernel.DaysBetween(from, to); got != 30 {
		t.Errorf("DaysBetween = %d, want 30", got)
	}

	if got := from.AddDays(30); got != to {
		t.Errorf("AddDays(30) = %s, want %s", got, to)
	}

	if !from.Before(to) || !to.After(from) || from.After(to) {
		t.Error("Before or After gives a wrong answer")
	}
}

func mustDate(t *testing.T, value string) kernel.Date {
	t.Helper()

	date, err := kernel.ParseDate(value)
	if err != nil {
		t.Fatal(err)
	}

	return date
}
