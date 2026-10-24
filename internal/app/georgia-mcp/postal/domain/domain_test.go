package domain_test

import (
	"errors"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/domain"
)

const (
	postalCodeLength = 4
	amberki          = "0186"
)

// POSTCODE-SC-009: A text of exactly 4 ASCII digits is a postal code, with its leading 0.
func TestParsePostalCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{input: amberki, want: amberki},
		{input: "6010", want: "6010"},
		{input: "186", wantErr: domain.ErrInvalidPostalCode},
		{input: "01860", wantErr: domain.ErrInvalidPostalCode},
		{input: "01a6", wantErr: domain.ErrInvalidPostalCode},
		{input: "٠١٨٦", wantErr: domain.ErrInvalidPostalCode},
		{input: " 0186", wantErr: domain.ErrInvalidPostalCode},
		{input: "", wantErr: domain.ErrInvalidPostalCode},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParsePostalCode(tt.input)
			if !errors.Is(err, tt.wantErr) || got.String() != tt.want {
				t.Errorf(
					"ParsePostalCode(%q) = %q, %v, want %q, %v",
					tt.input,
					got,
					err,
					tt.want,
					tt.wantErr,
				)
			}
		})
	}
}

// POSTCODE-SC-009: No input panics, and every accepted postal code has 4 ASCII digits.
func FuzzParsePostalCode(f *testing.F) {
	for _, seed := range []string{amberki, "186", "01a6", "٠١٨٦", "\x00\x00\x00\x00"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		code, err := domain.ParsePostalCode(input)
		if err != nil {
			return
		}

		digits := code.String()
		if len(digits) != postalCodeLength || digits != input {
			t.Fatalf("ParsePostalCode(%q) = %q, want the 4 digits of the input", input, digits)
		}

		for _, r := range digits {
			if r < '0' || r > '9' {
				t.Fatalf("ParsePostalCode(%q) = %q, want ASCII digits", input, digits)
			}
		}
	})
}
