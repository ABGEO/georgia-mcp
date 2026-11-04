package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	wagesRegion = "Social Statistics/Labour/Wages_Region.px"
	annualIndex = "105.9"
	flatIndex   = "100.0"
	smallFall   = "-0.5"
)

// GEOSTAT-SC-005: A figure keeps the text of Geostat. A missing figure is unpublished.
func TestParseFigure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input         string
		want          string
		wantPublished bool
		wantErr       error
	}{
		{input: annualIndex, want: annualIndex, wantPublished: true},
		{input: flatIndex, want: flatIndex, wantPublished: true},
		{input: smallFall, want: smallFall, wantPublished: true},
		{input: "1640", want: "1640", wantPublished: true},
		{input: ".."},
		{input: "..."},
		{input: "."},
		{input: "-"},
		{input: ""},
		{input: "1,5", wantErr: domain.ErrInvalidFigure},
		{input: "abc", wantErr: domain.ErrInvalidFigure},
		{input: "1e5", wantErr: domain.ErrInvalidFigure},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseFigure(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseFigure(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if got.String() != tt.want || got.Published() != tt.wantPublished {
				t.Errorf("ParseFigure(%q) = %q, %t, want %q, %t",
					tt.input, got, got.Published(), tt.want, tt.wantPublished)
			}
		})
	}
}

// GEOSTAT-SC-005: Minus100 subtracts 100 and keeps the count of decimals.
func TestFigureMinus100(t *testing.T) {
	t.Parallel()

	tests := []struct{ input, want string }{
		{annualIndex, "5.9"},
		{flatIndex, "0.0"},
		{"99.5", smallFall},
		{"112.34", "12.34"},
		{"..", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			figure, err := domain.ParseFigure(tt.input)
			if err != nil {
				t.Fatal(err)
			}

			if got := figure.Minus100().String(); got != tt.want {
				t.Errorf("Minus100(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// GEOSTAT-SC-005: No input panics, and a published figure gives back its text.
func FuzzParseFigure(f *testing.F) {
	for _, seed := range []string{annualIndex, "..", "-", "1,5", "1e5", "0.0"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		figure, err := domain.ParseFigure(input)
		if err != nil {
			return
		}

		if figure.Published() && figure.String() != input {
			t.Errorf("ParseFigure(%q).String() = %q", input, figure.String())
		}

		_ = figure.Minus100()
	})
}

// GEOSTAT-SC-006: A table code is a path of folders that ends with the file of a table.
func TestParseTableCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, input, wantFolder string
		wantErr                 error
	}{
		{name: "valid", input: wagesRegion, wantFolder: "Social Statistics/Labour"},
		{name: "root", input: "Wages.px", wantFolder: ""},
		{
			name: "capitals", input: "Population Census 2014/X/02TOTA~1.PX",
			wantFolder: "Population Census 2014/X",
		},
		{name: "only the extension", input: "a/.PX", wantErr: domain.ErrInvalidTableCode},
		{name: "empty", input: "", wantErr: domain.ErrInvalidTableCode},
		{name: "empty segment", input: "a//b.px", wantErr: domain.ErrInvalidTableCode},
		{name: "a slash first", input: "/a/b.px", wantErr: domain.ErrInvalidTableCode},
		{name: "a parent", input: "../x.px", wantErr: domain.ErrInvalidTableCode},
		{name: "a dot", input: "a/./x.px", wantErr: domain.ErrInvalidTableCode},
		{name: "no file", input: "Labour/Wages", wantErr: domain.ErrInvalidTableCode},
		{
			name: "long", input: strings.Repeat("a", 298) + ".px",
			wantErr: domain.ErrInvalidTableCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, err := domain.ParseTableCode(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseTableCode(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if err == nil && (code.String() != tt.input || code.Folder() != tt.wantFolder) {
				t.Errorf("ParseTableCode(%q) = %q in %q", tt.input, code, code.Folder())
			}
		})
	}
}

// GEOSTAT-SC-006: No input panics, and a valid code gives back its text.
func FuzzParseTableCode(f *testing.F) {
	for _, seed := range []string{wagesRegion, "../x.px", "a//b.px", "", ".px"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		code, err := domain.ParseTableCode(input)
		if err == nil && code.String() != input {
			t.Errorf("ParseTableCode(%q).String() = %q", input, code)
		}
	})
}

// GEOSTAT-SC-006: A month has the form YYYY-MM.
func TestParseMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		year    int
		month   int
		wantErr error
	}{
		{input: "2026-10", year: 2026, month: 10},
		{input: "2004-01", year: 2004, month: 1},
		{input: "2026-13", wantErr: domain.ErrInvalidMonth},
		{input: "2026-1", wantErr: domain.ErrInvalidMonth},
		{input: "2026-10-01", wantErr: domain.ErrInvalidMonth},
		{input: "October", wantErr: domain.ErrInvalidMonth},
		{input: "", wantErr: domain.ErrInvalidMonth},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			month, err := domain.ParseMonth(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseMonth(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if err == nil &&
				(month.Year() != tt.year || month.Number() != tt.month || month.String() != tt.input) {
				t.Errorf(
					"ParseMonth(%q) = %d, %d, %q",
					tt.input,
					month.Year(),
					month.Number(),
					month,
				)
			}
		})
	}
}

// GEOSTAT-SC-006: No input panics, and a valid month gives back its text.
func FuzzParseMonth(f *testing.F) {
	for _, seed := range []string{"2026-10", "2026-13", "0000-01", "-001-01"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		month, err := domain.ParseMonth(input)
		if err == nil && month.String() != input {
			t.Errorf("ParseMonth(%q).String() = %q", input, month)
		}
	})
}

// GEOSTAT-SC-006: A region has a code and a name in each language.
func TestParseRegion(t *testing.T) {
	t.Parallel()

	region, err := domain.ParseRegion("tbilisi")
	if err != nil || region.Name(kernel.LanguageEnglish) != "Tbilisi" ||
		region.Name(kernel.LanguageGeorgian) != "თბილისი" {
		t.Errorf("ParseRegion(tbilisi) = %q, %v", region, err)
	}

	if _, err := domain.ParseRegion("rustavi"); !errors.Is(err, domain.ErrUnknownRegion) {
		t.Errorf("ParseRegion(rustavi) error = %v, want ErrUnknownRegion", err)
	}

	if len(domain.Regions()) != 7 {
		t.Errorf("Regions() = %d, want 7", len(domain.Regions()))
	}
}

// GEOSTAT-SC-006: A price group has a code and a name in each language.
func TestParsePriceGroup(t *testing.T) {
	t.Parallel()

	group, err := domain.ParsePriceGroup("food")
	if err != nil || group.Name(kernel.LanguageEnglish) != "Food and non-alcoholic beverages" ||
		group.Name(kernel.LanguageGeorgian) != "სურსათი და უალკოჰოლო სასმელები" {
		t.Errorf("ParsePriceGroup(food) = %q, %v", group, err)
	}

	if _, err := domain.ParsePriceGroup("fuel"); !errors.Is(err, domain.ErrUnknownPriceGroup) {
		t.Errorf("ParsePriceGroup(fuel) error = %v, want ErrUnknownPriceGroup", err)
	}

	if len(domain.PriceGroups()) != 13 {
		t.Errorf("PriceGroups() = %d, want 13", len(domain.PriceGroups()))
	}
}

// GEOSTAT-SC-006: No input panics, and an accepted code gives back its text.
func FuzzParseRegionAndPriceGroup(f *testing.F) {
	for _, seed := range []string{"georgia", "total", "Tbilisi", ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if region, err := domain.ParseRegion(input); err == nil && string(region) != input {
			t.Errorf("ParseRegion(%q) = %q", input, region)
		}

		if group, err := domain.ParsePriceGroup(input); err == nil && string(group) != input {
			t.Errorf("ParsePriceGroup(%q) = %q", input, group)
		}
	})
}
