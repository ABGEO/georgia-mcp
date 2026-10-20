package domain

import (
	"fmt"
	"strings"
)

const (
	minCardCodeLength = 2
	maxCardCodeLength = 40
)

// CardCode is the short name of a card in a request: 2 to 40 lower case letters,
// digits, and underscores. Each bank has its own codes.
type CardCode string

// ParseCardCode returns the code of a card.
func ParseCardCode(text string) (CardCode, error) {
	if len(text) < minCardCodeLength || len(text) > maxCardCodeLength {
		return "", fmt.Errorf("%w: %q", ErrInvalidCardCode, text)
	}

	for _, r := range text {
		if !isCardCodeRune(r) {
			return "", fmt.Errorf("%w: %q", ErrInvalidCardCode, text)
		}
	}

	return CardCode(text), nil
}

// CardCodeOf returns the card code of a code of a bank that no table holds: the lower
// case text, with each character that a card code does not allow replaced by "_".
func CardCodeOf(vendorCode string) CardCode {
	return CardCode(strings.Map(func(char rune) rune {
		if char >= 'A' && char <= 'Z' {
			char += 'a' - 'A'
		}

		if isCardCodeRune(char) {
			return char
		}

		return '_'
	}, vendorCode))
}

func isCardCodeRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
}

// Card is a card or a package of a bank that an offer names as a condition.
type Card struct {
	Code CardCode
	Name string
}
