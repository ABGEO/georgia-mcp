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

// BankKind is the kind of a company that publishes commercial rates.
type BankKind string

const (
	// KindBank is a commercial bank.
	KindBank BankKind = "bank"
	// KindMicrofinance is a microfinance organization: the NBG registers it to lend
	// money, and it is not a bank.
	KindMicrofinance BankKind = "microfinance organization"
	// KindExchangeBureau is an online exchange bureau: the NBG registers it to change
	// money, and it changes money by bank transfer only.
	KindExchangeBureau BankKind = "online exchange bureau"
)

// Bank is a company that publishes commercial rates. Kind tells a bank from a company
// that is not one. An empty Kind is a bank.
type Bank struct {
	Code BankCode
	Name string
	Kind BankKind
}
