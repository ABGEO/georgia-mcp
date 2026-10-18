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

	code := strings.ToUpper(text)
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return "", fmt.Errorf("%w: %q", ErrInvalidCurrencyCode, text)
		}
	}

	return CurrencyCode(code), nil
}
