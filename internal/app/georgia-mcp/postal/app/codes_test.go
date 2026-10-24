package app_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func readAt() time.Time {
	return time.Date(2026, 10, 24, 12, 0, 0, 0, kernel.TbilisiZone())
}

// fakeSource answers fixed addresses and records each call.
type fakeSource struct {
	addresses []domain.Address
	texts     []string
	languages []kernel.Language
}

func (f *fakeSource) Addresses(
	_ context.Context,
	text string,
	language kernel.Language,
) (app.AddressList, error) {
	f.texts = append(f.texts, text)
	f.languages = append(f.languages, language)

	return app.AddressList{Addresses: f.addresses, ReadAt: readAt(), Stale: true}, nil
}

// addresses returns count addresses of Tbilisi with distinct texts.
func addresses(t *testing.T, count int) []domain.Address {
	t.Helper()

	code, err := domain.ParsePostalCode("0179")
	if err != nil {
		t.Fatal(err)
	}

	list := make([]domain.Address, count)
	for index := range list {
		list[index] = domain.Address{
			Text:     "ქალაქი თბილისი, " + strings.Repeat("ა", index+1),
			District: "თბილისი",
			Code:     code,
		}
	}

	return list
}

// POSTCODE-SC-011: A text of fewer than 3 or more than 100 characters, after the spaces
// at the ends, is refused before a call. The source gets the text with each run of white
// space as one space.
func TestSearchCodesChecksTheText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		wantText string
		wantErr  error
	}{
		{name: "two letters", text: "ab", wantErr: app.ErrInvalidQuery},
		{name: "two letters in spaces", text: "  ab  ", wantErr: app.ErrInvalidQuery},
		{name: "101 letters", text: strings.Repeat("ა", 101), wantErr: app.ErrInvalidQuery},
		{name: "100 letters", text: strings.Repeat("ა", 100), wantText: strings.Repeat("ა", 100)},
		{name: "four letters", text: "ვაკე", wantText: "ვაკე"},
		{name: "inner spaces", text: "  ილია   ჭავჭავაძის ", wantText: "ილია ჭავჭავაძის"},
		{name: "case", text: "Rustavi", wantText: "Rustavi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := &fakeSource{}

			_, err := app.NewSearchCodes(source).Search(t.Context(), app.CodeQuery{
				Text: tt.text, Language: kernel.LanguageGeorgian, Limit: 20,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Search() error = %v, want %v", err, tt.wantErr)
			}

			var wantTexts []string
			if tt.wantErr == nil {
				wantTexts = []string{tt.wantText}
			}

			if !slices.Equal(source.texts, wantTexts) {
				t.Errorf("source texts = %q, want %q", source.texts, wantTexts)
			}
		})
	}
}

// POSTCODE-SC-012: The use case keeps the order of the source, cuts the list to the
// limit, and counts every match.
func TestSearchCodesCutsToTheLimit(t *testing.T) {
	t.Parallel()

	all := addresses(t, 63)

	tests := []struct {
		limit int
		want  []domain.Address
	}{
		{limit: 20, want: all[:20]},
		{limit: 100, want: all},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.limit), func(t *testing.T) {
			t.Parallel()

			source := &fakeSource{addresses: all}

			got, err := app.NewSearchCodes(source).Search(t.Context(), app.CodeQuery{
				Text: "ჭავჭავაძის", Language: kernel.LanguageEnglish, Limit: tt.limit,
			})
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if !slices.Equal(got.Addresses, tt.want) {
				t.Errorf(
					"Search() addresses = %d in some order, want the first %d",
					len(got.Addresses),
					len(tt.want),
				)
			}

			if got.Total != len(all) {
				t.Errorf("Search() total = %d, want %d", got.Total, len(all))
			}

			if !got.ReadAt.Equal(readAt()) || !got.Stale {
				t.Errorf(
					"Search() read = %v, stale %t, want %v, true",
					got.ReadAt,
					got.Stale,
					readAt(),
				)
			}

			if !slices.Equal(source.languages, []kernel.Language{kernel.LanguageEnglish}) {
				t.Errorf("source languages = %q, want [en]", source.languages)
			}
		})
	}
}
