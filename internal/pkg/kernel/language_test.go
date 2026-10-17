package kernel_test

import (
	"errors"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func TestParseLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    kernel.Language
		wantErr error
	}{
		{name: "georgian", input: "ka", want: kernel.LanguageGeorgian},
		{name: "english", input: "en", want: kernel.LanguageEnglish},
		{name: "empty gives the default", input: "", want: kernel.LanguageEnglish},
		{name: "upper case", input: "KA", wantErr: kernel.ErrUnknownLanguage},
		{name: "unknown code", input: "ru", wantErr: kernel.ErrUnknownLanguage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseLanguage(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseLanguage(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("ParseLanguage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func FuzzParseLanguage(f *testing.F) {
	for _, seed := range []string{"", "ka", "en", "KA", "ქა", "\x00"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got, err := kernel.ParseLanguage(input)
		if err != nil {
			if !errors.Is(err, kernel.ErrUnknownLanguage) {
				t.Fatalf("ParseLanguage(%q) error = %v, want ErrUnknownLanguage", input, err)
			}

			return
		}

		if got != kernel.LanguageGeorgian && got != kernel.LanguageEnglish {
			t.Fatalf("ParseLanguage(%q) = %q, a value that is not valid", input, got)
		}
	})
}
