package domain

import "fmt"

const postalCodeLength = 4

// PostalCode is a postal code of Georgian Post: 4 ASCII digits. A leading 0 stays.
type PostalCode struct {
	code string
}

// ParsePostalCode returns the postal code of a text of exactly 4 ASCII digits.
func ParsePostalCode(text string) (PostalCode, error) {
	if len(text) != postalCodeLength {
		return PostalCode{}, fmt.Errorf("%w: %q", ErrInvalidPostalCode, text)
	}

	for _, r := range text {
		if r < '0' || r > '9' {
			return PostalCode{}, fmt.Errorf("%w: %q", ErrInvalidPostalCode, text)
		}
	}

	return PostalCode{code: text}, nil
}

// String returns the 4 digits: "0186".
func (c PostalCode) String() string {
	return c.code
}
