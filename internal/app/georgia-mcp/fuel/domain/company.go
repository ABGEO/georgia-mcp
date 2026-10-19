package domain

import "fmt"

const (
	minCompanyCodeLength = 2
	maxCompanyCodeLength = 16
)

// CompanyCode is the short name of a fuel company in a request: 2 to 16 lower case letters.
type CompanyCode string

// ParseCompanyCode returns the code of a fuel company. It accepts lower case letters only.
func ParseCompanyCode(text string) (CompanyCode, error) {
	if len(text) < minCompanyCodeLength || len(text) > maxCompanyCodeLength {
		return "", fmt.Errorf("%w: %q", ErrInvalidCompanyCode, text)
	}

	for _, r := range text {
		if r < 'a' || r > 'z' {
			return "", fmt.Errorf("%w: %q", ErrInvalidCompanyCode, text)
		}
	}

	return CompanyCode(text), nil
}

// Company is a company that sells fuel at its own network of stations in Georgia.
// PublishesEffectiveDate is false when the company gives no effective date to its
// prices.
type Company struct {
	Code                   CompanyCode
	Name                   string
	PublishesEffectiveDate bool
}
