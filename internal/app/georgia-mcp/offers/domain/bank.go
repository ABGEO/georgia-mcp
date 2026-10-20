package domain

import "fmt"

const (
	minBankCodeLength = 2
	maxBankCodeLength = 16
)

// BankCode is the short name of a bank in a request: 2 to 16 lower case letters.
type BankCode string

// ParseBankCode returns the code of a bank. It accepts lower case letters only.
func ParseBankCode(text string) (BankCode, error) {
	if len(text) < minBankCodeLength || len(text) > maxBankCodeLength {
		return "", fmt.Errorf("%w: %q", ErrInvalidBankCode, text)
	}

	for _, r := range text {
		if r < 'a' || r > 'z' {
			return "", fmt.Errorf("%w: %q", ErrInvalidBankCode, text)
		}
	}

	return BankCode(text), nil
}

// Bank is a commercial bank of Georgia that publishes card offers.
type Bank struct {
	Code BankCode
	Name string
}
