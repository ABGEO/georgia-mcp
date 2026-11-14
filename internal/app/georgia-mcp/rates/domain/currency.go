package domain

import (
	"fmt"
	"strings"
)

const currencyCodeLength = 3

// CurrencyCode is an ISO 4217 code of three upper case letters.
type CurrencyCode string

// ParseCurrencyCode returns the code of three letters in any case, in upper case.
func ParseCurrencyCode(text string) (CurrencyCode, error) {
	if len(text) != currencyCodeLength {
		return "", fmt.Errorf("%w: %q", ErrInvalidCurrencyCode, text)
	}

	for i := range len(text) {
		if c := text[i]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return "", fmt.Errorf("%w: %q", ErrInvalidCurrencyCode, text)
		}
	}

	return CurrencyCode(strings.ToUpper(text)), nil
}
