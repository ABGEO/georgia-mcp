package domain_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	wissol          = "wissol"
	regular         = "Euro Regular"
	regularGeorgian = "ევრო რეგულარი"
	kerosene        = "kerosene"
)

// WISFUEL-SC-004: A company code has lower case letters only, 2 to 16 of them.
func TestParseCompanyCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    domain.CompanyCode
		wantErr error
	}{
		{input: wissol, want: wissol},
		{input: "Wissol", wantErr: domain.ErrInvalidCompanyCode},
		{input: "w", wantErr: domain.ErrInvalidCompanyCode},
		{input: "wis-sol", wantErr: domain.ErrInvalidCompanyCode},
		{input: "abcdefghijklmnopq", wantErr: domain.ErrInvalidCompanyCode},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseCompanyCode(tt.input)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Errorf(
					"ParseCompanyCode(%q) = %q, %v, want %q, %v",
					tt.input, got, err, tt.want, tt.wantErr,
				)
			}
		})
	}
}

// WISFUEL-SC-004: No input panics, and every accepted code has lower case letters only.
func FuzzParseCompanyCode(f *testing.F) {
	for _, seed := range []string{wissol, "Wissol", "w", "wis-sol", "ვისოლი"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		code, err := domain.ParseCompanyCode(input)
		if err != nil {
			return
		}

		for _, r := range code {
			if r < 'a' || r > 'z' {
				t.Fatalf("ParseCompanyCode(%q) = %q, want lower case letters", input, code)
			}
		}
	})
}

// WISFUEL-SC-005: A grade is an exact lower case value of the list, and gives its unit.
func TestParseGrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		want     domain.Grade
		wantUnit domain.Unit
		wantErr  error
	}{
		{input: "diesel", want: domain.GradeDiesel, wantUnit: domain.UnitLitre},
		{input: "cng", want: domain.GradeCNG, wantUnit: domain.UnitCubicMetre},
		{input: "super_100", want: domain.GradeSuper100, wantUnit: domain.UnitLitre},
		{input: "Diesel", wantErr: domain.ErrInvalidGrade},
		{input: kerosene, wantErr: domain.ErrInvalidGrade},
		{input: "", wantErr: domain.ErrInvalidGrade},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseGrade(tt.input)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Fatalf(
					"ParseGrade(%q) = %q, %v, want %q, %v",
					tt.input, got, err, tt.want, tt.wantErr,
				)
			}

			if tt.wantErr == nil && got.Unit() != tt.wantUnit {
				t.Errorf("%s.Unit() = %q, want %q", got, got.Unit(), tt.wantUnit)
			}
		})
	}
}

// WISFUEL-SC-005: No input panics, and every accepted grade is in the list of grades.
func FuzzParseGrade(f *testing.F) {
	for _, seed := range []string{"diesel", "cng", "super_100", "Diesel", kerosene, ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		grade, err := domain.ParseGrade(input)
		if err != nil {
			return
		}

		if string(grade) != input {
			t.Fatalf("ParseGrade(%q) = %q", input, grade)
		}
	})
}

// SOCFUEL-SC-004: The grade lpg is the exact lower case value, for one litre.
func TestParseGradeOfLPG(t *testing.T) {
	t.Parallel()

	got, err := domain.ParseGrade("lpg")
	if err != nil || got != domain.GradeLPG || got.Unit() != domain.UnitLitre {
		t.Errorf("ParseGrade(lpg) = %q (%q), %v, want lpg (litre)", got, got.Unit(), err)
	}

	if _, err := domain.ParseGrade("LPG"); !errors.Is(err, domain.ErrInvalidGrade) {
		t.Errorf("ParseGrade(LPG) error = %v, want ErrInvalidGrade", err)
	}
}

// WISFUEL-SC-005, SOCFUEL-SC-004: The list of grades has the order of the requirements.
func TestGrades(t *testing.T) {
	t.Parallel()

	want := []domain.Grade{
		domain.GradeRegular, domain.GradePremium, domain.GradeSuper, domain.GradeSuper100,
		domain.GradeDiesel, domain.GradeCNG, domain.GradeLPG, domain.GradeOther,
	}

	got := domain.Grades()
	if len(got) != len(want) {
		t.Fatalf("Grades() = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Grades()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// WISFUEL-SC-006: A product price needs a standard price more than zero, and a
// self-service price more than zero when it has one.
func TestNewProductPrice(t *testing.T) {
	t.Parallel()

	effective, err := kernel.ParseDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		standard    string
		selfService decimal.NullDecimal
		wantErr     error
	}{
		{name: "both prices", standard: "4.08", selfService: valid("3.93")},
		{name: "no self-service price", standard: "4.79"},
		{name: "zero standard", standard: "0", wantErr: domain.ErrInvalidProductPrice},
		{
			name:        "zero self-service",
			standard:    "4.08",
			selfService: valid("0"),
			wantErr:     domain.ErrInvalidProductPrice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewProductPrice(
				"05",
				domain.ProductName{English: regular},
				domain.GradeRegular,
				decimal.RequireFromString(tt.standard),
				tt.selfService,
				effective,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewProductPrice() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && got.SelfService.Valid != tt.selfService.Valid {
				t.Errorf(
					"SelfService.Valid = %v, want %v",
					got.SelfService.Valid,
					tt.selfService.Valid,
				)
			}
		})
	}
}

// WISFUEL-SC-006: A product price needs a code, a name, and a known grade.
func TestNewProductPriceRefusesAnIncompleteProduct(t *testing.T) {
	t.Parallel()

	effective, err := kernel.ParseDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}

	name := domain.ProductName{English: regular}
	price := decimal.RequireFromString("4.08")

	tests := []struct {
		name      string
		code      string
		product   domain.ProductName
		grade     domain.Grade
		effective kernel.Date
	}{
		{name: "no code", product: name, grade: domain.GradeRegular, effective: effective},
		{name: "no name", code: "05", grade: domain.GradeRegular, effective: effective},
		{name: "unknown grade", code: "05", product: name, grade: kerosene, effective: effective},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewProductPrice(
				tt.code, tt.product, tt.grade, price, decimal.NullDecimal{}, tt.effective,
			)
			if !errors.Is(err, domain.ErrInvalidProductPrice) {
				t.Errorf("NewProductPrice() error = %v, want ErrInvalidProductPrice", err)
			}
		})
	}
}

// ROMFUEL-SC-005: A product price takes a date, or the zero date of a company that
// publishes none.
func TestNewProductPriceTakesTheZeroDate(t *testing.T) {
	t.Parallel()

	effective, err := kernel.ParseDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		effective kernel.Date
	}{
		{name: "a date", effective: effective},
		{name: "the zero date", effective: kernel.Date{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewProductPrice(
				"efix_super",
				domain.ProductName{Georgian: "efix სუპერი"},
				domain.GradeSuper,
				decimal.RequireFromString("4.65"),
				decimal.NullDecimal{},
				tt.effective,
			)
			if err != nil {
				t.Fatalf("NewProductPrice() error = %v", err)
			}

			if got.EffectiveDate != tt.effective {
				t.Errorf("EffectiveDate = %v, want %v", got.EffectiveDate, tt.effective)
			}
		})
	}
}

// WISFUEL-SC-007: A name is in the requested language, or in the other one when it lacks it.
func TestProductNameIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		product  domain.ProductName
		language kernel.Language
		want     string
	}{
		{
			name:     "georgian",
			product:  domain.ProductName{Georgian: regularGeorgian, English: regular},
			language: kernel.LanguageGeorgian,
			want:     regularGeorgian,
		},
		{
			name:     "english",
			product:  domain.ProductName{Georgian: regularGeorgian, English: regular},
			language: kernel.LanguageEnglish,
			want:     regular,
		},
		{
			name:     "no english",
			product:  domain.ProductName{Georgian: regularGeorgian},
			language: kernel.LanguageEnglish,
			want:     regularGeorgian,
		},
		{
			name:     "no georgian",
			product:  domain.ProductName{English: regular},
			language: kernel.LanguageGeorgian,
			want:     regular,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.product.In(tt.language); got != tt.want {
				t.Errorf("In(%s) = %q, want %q", tt.language, got, tt.want)
			}
		})
	}
}

func valid(value string) decimal.NullDecimal {
	return decimal.NewNullDecimal(decimal.RequireFromString(value))
}
