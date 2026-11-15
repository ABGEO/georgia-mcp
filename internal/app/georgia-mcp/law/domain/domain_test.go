package domain_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const inserted = "133^3"

// MATSNE-SC-006: An article number is a base and an optional insert, written with ^.
func TestParseArticleNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "82", want: "82"},
		{input: inserted, want: inserted},
		{input: "133³", want: inserted},
		{input: " 9999^12 ", want: "9999^12"},
		{input: "82a", wantErr: true},
		{input: "0", wantErr: true},
		{input: "082", wantErr: true},
		{input: "133^", wantErr: true},
		{input: "^3", wantErr: true},
		{input: "133^0", wantErr: true},
		{input: "12345", wantErr: true},
		{input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseArticleNumber(tt.input)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidArticleNumber) {
					t.Fatalf(
						"ParseArticleNumber(%q) error = %v, want ErrInvalidArticleNumber",
						tt.input,
						err,
					)
				}

				return
			}

			if err != nil || got.String() != tt.want {
				t.Fatalf("ParseArticleNumber(%q) = %q, %v, want %q", tt.input, got, err, tt.want)
			}
		})
	}
}

// MATSNE-SC-006: The text of a parsed article number parses to the same value.
func FuzzParseArticleNumber(f *testing.F) {
	for _, seed := range []string{"82", inserted, "133³", "0", "^", "1^²³", "٣"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		number, err := domain.ParseArticleNumber(input)
		if err != nil {
			return
		}

		again, err := domain.ParseArticleNumber(number.String())
		if err != nil || again != number {
			t.Errorf(
				"ParseArticleNumber(%q) = %q, which parses to %q, %v",
				input,
				number,
				again,
				err,
			)
		}
	})
}

// MATSNE-SC-007: An act identifier is 1 to 10 digits with no leading zero.
func TestParseActID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		wantErr bool
	}{
		{input: "1043717"},
		{input: "9999999999"},
		{input: "abc", wantErr: true},
		{input: "0", wantErr: true},
		{input: "01", wantErr: true},
		{input: "10437170000", wantErr: true},
		{input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseActID(tt.input)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidActID) {
					t.Fatalf("ParseActID(%q) error = %v, want ErrInvalidActID", tt.input, err)
				}

				return
			}

			if err != nil || got.String() != tt.input {
				t.Fatalf("ParseActID(%q) = %q, %v", tt.input, got, err)
			}
		})
	}
}

// MATSNE-SC-007: A parsed act identifier is digits with no leading zero.
func FuzzParseActID(f *testing.F) {
	for _, seed := range []string{"1043717", "0", "01", "١٢", ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		id, err := domain.ParseActID(input)
		if err != nil {
			return
		}

		text := id.String()
		if text == "" || text[0] == '0' || len(text) > 10 {
			t.Errorf("ParseActID(%q) = %q", input, text)
		}

		for _, r := range text {
			if r < '0' || r > '9' {
				t.Errorf("ParseActID(%q) = %q, not ASCII digits", input, text)
			}
		}
	})
}

// MATSNE-SC-007: Each of the 16 document types parses, and an unknown one does not.
func TestParseDocumentType(t *testing.T) {
	t.Parallel()

	types := domain.DocumentTypes()
	if len(types) != 16 || types[0] != "law" || types[15] != "draft" {
		t.Fatalf("DocumentTypes() = %q, want 16 from law to draft", types)
	}

	for _, code := range types {
		if got, err := domain.ParseDocumentType(string(code)); err != nil || got != code {
			t.Errorf("ParseDocumentType(%q) = %q, %v", code, got, err)
		}
	}

	if _, err := domain.ParseDocumentType(
		"statute",
	); !errors.Is(
		err,
		domain.ErrUnknownDocumentType,
	) {
		t.Errorf("ParseDocumentType(statute) error = %v, want ErrUnknownDocumentType", err)
	}
}

// MATSNE-SC-007: A parsed document type is one of the 16.
func FuzzParseDocumentType(f *testing.F) {
	for _, seed := range []string{"law", "LAW", "statute", ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got, err := domain.ParseDocumentType(input)
		if err == nil && !slices.Contains(domain.DocumentTypes(), got) {
			t.Errorf("ParseDocumentType(%q) = %q, not a document type", input, got)
		}
	})
}

// MATSNE-SC-007: The three statuses parse. The name of the source does not.
func TestParseActStatus(t *testing.T) {
	t.Parallel()

	for _, status := range []domain.ActStatus{domain.StatusActive, domain.StatusExpired, domain.StatusPending} {
		if got, err := domain.ParseActStatus(string(status)); err != nil || got != status {
			t.Errorf("ParseActStatus(%q) = %q, %v", status, got, err)
		}
	}

	if _, err := domain.ParseActStatus(
		"toBeActivated",
	); !errors.Is(
		err,
		domain.ErrUnknownActStatus,
	) {
		t.Errorf("ParseActStatus(toBeActivated) error = %v, want ErrUnknownActStatus", err)
	}
}

// MATSNE-SC-007: A parsed status is one of the three.
func FuzzParseActStatus(f *testing.F) {
	for _, seed := range []string{"active", "Expired", "toBeActivated", ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got, err := domain.ParseActStatus(input)
		known := []domain.ActStatus{domain.StatusActive, domain.StatusExpired, domain.StatusPending}

		if err == nil && !slices.Contains(known, got) {
			t.Errorf("ParseActStatus(%q) = %q, not a status", input, got)
		}
	})
}

func date(t *testing.T, text string) kernel.Date {
	t.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// MATSNE-SC-008: VersionOn gives the newest version on or before a date, the higher
// number on a tie.
func TestVersionOn(t *testing.T) {
	t.Parallel()

	act := domain.Act{Versions: []domain.Version{
		{Number: 4, Date: date(t, "2026-10-28")},
		{Number: 3, Date: date(t, "2026-04-01")},
		{Number: 2, Date: date(t, "2026-04-01")},
		{Number: 1, Date: date(t, "2011-01-01")},
		{Number: 0, Date: date(t, "2010-10-12")},
	}}

	tests := []struct {
		on     string
		want   int
		wantOK bool
	}{
		{on: "2025-01-01", want: 1, wantOK: true},
		{on: "2026-04-01", want: 3, wantOK: true},
		{on: "2027-01-01", want: 4, wantOK: true},
		{on: "2010-10-12", want: 0, wantOK: true},
		{on: "2010-10-11", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.on, func(t *testing.T) {
			t.Parallel()

			got, ok := act.VersionOn(date(t, tt.on))
			if ok != tt.wantOK || (ok && got.Number != tt.want) {
				t.Errorf("VersionOn(%s) = %+v, %v, want %d, %v", tt.on, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
