package domain

import (
	"fmt"
	"strings"
)

const (
	minGroups      = 4
	maxGroups      = 5
	maxGroupDigits = 4
)

// CadastralCode is the code of a parcel: 4 or 5 groups of 1 to 4 digits, with dots.
type CadastralCode struct {
	value string
}

// ParseCadastralCode trims the text. A code of a unit, with 6 groups or more, fails.
func ParseCadastralCode(text string) (CadastralCode, error) {
	text = strings.TrimSpace(text)
	groups := strings.Split(text, ".")

	if len(groups) < minGroups || len(groups) > maxGroups {
		return CadastralCode{}, fmt.Errorf("%w: %q", ErrInvalidCadastralCode, text)
	}

	for _, group := range groups {
		if group == "" || len(group) > maxGroupDigits || !isDigits(group) {
			return CadastralCode{}, fmt.Errorf("%w: %q", ErrInvalidCadastralCode, text)
		}
	}

	return CadastralCode{value: text}, nil
}

// String returns the code: "01.18.04.020.005".
func (c CadastralCode) String() string {
	return c.value
}

// IsZero reports whether the code is absent.
func (c CadastralCode) IsZero() bool {
	return c.value == ""
}

func isDigits(text string) bool {
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
