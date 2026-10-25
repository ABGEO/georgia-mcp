package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const galleria2215 = "cag_5/27-10-2026/22:15"

// CAVCINE-SC-010: A session code gives its hall key and its start in Tbilisi. A code of
// another form is invalid.
func TestParseSessionCode(t *testing.T) {
	t.Parallel()

	code, err := domain.ParseSessionCode(galleria2215)
	if err != nil {
		t.Fatalf("ParseSessionCode(%q) error = %v", galleria2215, err)
	}

	want := time.Date(2026, time.October, 27, 22, 15, 0, 0, kernel.TbilisiZone())
	if !code.Start().Equal(want) ||
		code.Start().Format(time.RFC3339) != "2026-10-27T22:15:00+04:00" {
		t.Errorf("Start() = %v, want %v", code.Start(), want)
	}

	if code.HallKey() != "cag_5" || code.String() != galleria2215 ||
		code.Date().String() != "2026-10-27" {
		t.Errorf("code = %q, hall key %q, date %v", code, code.HallKey(), code.Date())
	}

	if again, _ := domain.ParseSessionCode(galleria2215); again != code {
		t.Errorf("two codes of one text differ: %+v and %+v", again, code)
	}

	for _, input := range []string{
		"cag_5/32-10-2026/22:15",
		"cag_5/27-10-2026",
		"cag_5/27-10-2026/24:00",
		"cag5/27-10-2026/22:15",
		"cag_5/27-10-2026/22:15/x",
		"xyz",
		"",
	} {
		if _, err := domain.ParseSessionCode(input); !errors.Is(err, domain.ErrInvalidSessionCode) {
			t.Errorf("ParseSessionCode(%q) error = %v, want ErrInvalidSessionCode", input, err)
		}
	}
}

// CAVCINE-SC-010: No input panics, and every accepted code starts in Tbilisi and writes
// back to its text.
func FuzzParseSessionCode(f *testing.F) {
	for _, seed := range []string{galleria2215, "ccm_x/27-10-2026/21:45", "cag_5/32-10-2026/22:15", "/"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		code, err := domain.ParseSessionCode(input)
		if err != nil {
			return
		}

		if _, offset := code.Start().Zone(); offset != 4*60*60 {
			t.Errorf("ParseSessionCode(%q) start offset = %d, want +04:00", input, offset)
		}

		if code.String() != input {
			t.Errorf("ParseSessionCode(%q).String() = %q", input, code)
		}
	})
}

// CAVCINE-SC-014: A price is greater than zero, and a seat class is one of the four.
func TestValuesThatFailAnAnswer(t *testing.T) {
	t.Parallel()

	if price, err := domain.NewPrice(
		"20",
	); err != nil ||
		!price.Amount().Equal(decimal.NewFromInt(20)) {
		t.Errorf("NewPrice(20) = %v, %v, want 20", price, err)
	}

	for _, text := range []string{"0", "-1", "", "abc"} {
		if _, err := domain.NewPrice(text); !errors.Is(err, domain.ErrInvalidPrice) {
			t.Errorf("NewPrice(%q) error = %v, want ErrInvalidPrice", text, err)
		}
	}

	if _, err := domain.ParseSeatClass("gold"); !errors.Is(err, domain.ErrUnknownSeatClass) {
		t.Errorf("ParseSeatClass(gold) error = %v, want ErrUnknownSeatClass", err)
	}

	if _, err := domain.ParseAudioLanguage("ru"); !errors.Is(err, domain.ErrUnknownAudioLanguage) {
		t.Errorf("ParseAudioLanguage(ru) error = %v, want ErrUnknownAudioLanguage", err)
	}
}

// CAVCINE-SC-016: A code of a request has the form of its kind.
func TestParseCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		parse func(string) error
		valid []string
		bad   []string
		want  error
	}{
		{
			name: "cinema",
			parse: func(text string) error {
				_, err := domain.ParseCinemaCode(text)

				return err
			},
			valid: []string{"east_point", "city_mall_saburtalo"},
			bad:   []string{"", "East Point", "east-point"},
			want:  domain.ErrInvalidCinemaCode,
		},
		{
			name: "movie",
			parse: func(text string) error {
				_, err := domain.ParseMovieCode(text)

				return err
			},
			valid: []string{"street-fighter", "clayface-01"},
			bad:   []string{"", "Street Fighter", "street_fighter"},
			want:  domain.ErrInvalidMovieCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, text := range tt.valid {
				if err := tt.parse(text); err != nil {
					t.Errorf("parse(%q) error = %v", text, err)
				}
			}

			for _, text := range tt.bad {
				if err := tt.parse(text); !errors.Is(err, tt.want) {
					t.Errorf("parse(%q) error = %v, want %v", text, err, tt.want)
				}
			}
		})
	}
}

// CAVCINE-SC-019: A text gives the language of the request, and the other language when
// it has none.
func TestTextIn(t *testing.T) {
	t.Parallel()

	name := domain.Text{Georgian: "კავეა ისთ ფოინთი", English: "Cavea East Point"}
	if name.In(kernel.LanguageGeorgian) != "კავეა ისთ ფოინთი" ||
		name.In(kernel.LanguageEnglish) != "Cavea East Point" {
		t.Errorf(
			"In() = %q and %q",
			name.In(kernel.LanguageGeorgian),
			name.In(kernel.LanguageEnglish),
		)
	}

	if only := (domain.Text{English: "Cavea"}); only.In(kernel.LanguageGeorgian) != "Cavea" {
		t.Errorf("In(ka) of an English text = %q, want Cavea", only.In(kernel.LanguageGeorgian))
	}
}
