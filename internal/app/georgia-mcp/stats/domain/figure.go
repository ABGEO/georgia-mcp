package domain

import (
	"fmt"
	"regexp"

	"github.com/shopspring/decimal"
)

// decimalText is the form of a figure of Geostat: no exponent, no thousands separator.
var decimalText = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

const hundred = 100

// Figure is one figure of a table as Geostat publishes it, or a figure that Geostat did
// not publish. The zero value is an unpublished figure.
type Figure struct {
	value     decimal.Decimal
	text      string
	published bool
}

// ParseFigure returns the figure of a text of Geostat. PxWeb marks a figure that Geostat
// did not publish with "..", "...", ".", "-", or no text.
func ParseFigure(text string) (Figure, error) {
	switch text {
	case "", ".", "..", "...", "-":
		return Figure{}, nil
	}

	if !decimalText.MatchString(text) {
		return Figure{}, fmt.Errorf("%w: %q", ErrInvalidFigure, text)
	}

	value, err := decimal.NewFromString(text)
	if err != nil {
		return Figure{}, fmt.Errorf("%w: %q", ErrInvalidFigure, text)
	}

	return Figure{value: value, text: text, published: true}, nil
}

// Published tells whether Geostat published the figure.
func (f Figure) Published() bool {
	return f.published
}

// String returns the text of Geostat: "105.9". An unpublished figure gives "".
func (f Figure) String() string {
	return f.text
}

// Minus100 returns the figure minus 100, with the count of decimals of the figure: an
// index of 105.9 gives the change 5.9. An unpublished figure stays unpublished.
func (f Figure) Minus100() Figure {
	if !f.published {
		return Figure{}
	}

	value := f.value.Sub(decimal.NewFromInt(hundred))
	decimals := max(-f.value.Exponent(), 0)

	return Figure{value: value, text: value.StringFixed(decimals), published: true}
}
