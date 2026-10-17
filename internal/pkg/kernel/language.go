package kernel

import "fmt"

// Language is the language of a text that a source publishes.
type Language string

const (
	// LanguageGeorgian is Georgian.
	LanguageGeorgian Language = "ka"
	// LanguageEnglish is English.
	LanguageEnglish Language = "en"
)

// ParseLanguage returns the language of a code. The empty code gives English.
func ParseLanguage(code string) (Language, error) {
	switch Language(code) {
	case "", LanguageEnglish:
		return LanguageEnglish, nil
	case LanguageGeorgian:
		return LanguageGeorgian, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownLanguage, code)
	}
}
