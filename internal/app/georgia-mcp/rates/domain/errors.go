package domain

import "errors"

var (
	// ErrInvalidCurrencyCode tells that a text is not three letters.
	ErrInvalidCurrencyCode = errors.New("currency: invalid code")
	// ErrInvalidDate tells that a text is not a calendar date in YYYY-MM-DD.
	ErrInvalidDate = errors.New("date: invalid")
	// ErrInvalidQuantity tells that a quantity is not a power of ten.
	ErrInvalidQuantity = errors.New("rate: invalid quantity")
)
