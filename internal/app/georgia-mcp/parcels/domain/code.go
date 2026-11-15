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

// ParseCadastralCode trims the text. 6 or more groups of digits give a *UnitCodeError.
func ParseCadastralCode(text string) (CadastralCode, error) {
	text = strings.TrimSpace(text)
	groups := strings.Split(text, ".")

	for _, group := range groups {
		if !isDigits(group) {
			return CadastralCode{}, fmt.Errorf("%w: %q", ErrInvalidCadastralCode, text)
		}
	}

	switch {
	case len(groups) > maxGroups:
		parcel := CadastralCode{value: strings.Join(groups[:maxGroups], ".")}

		return CadastralCode{}, &UnitCodeError{Parcel: parcel}
	case len(groups) < minGroups:
		return CadastralCode{}, fmt.Errorf("%w: %q", ErrInvalidCadastralCode, text)
	}

	return CadastralCode{value: text}, nil
}

// String returns the code: "01.15.05.066.020".
func (c CadastralCode) String() string {
	return c.value
}

func isDigits(group string) bool {
	if group == "" || len(group) > maxGroupDigits {
		return false
	}

	for _, r := range group {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
