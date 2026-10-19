package domain

import "errors"

var (
	// ErrInvalidBankCode tells that a text is not 2 to 16 lower case letters.
	ErrInvalidBankCode = errors.New("bank: invalid code")
	// ErrInvalidCommercialRate tells that a buy or a sell price is not more than zero.
	ErrInvalidCommercialRate = errors.New("rate: invalid commercial rate")
	// ErrInvalidCurrencyCode tells that a text is not three letters.
	ErrInvalidCurrencyCode = errors.New("currency: invalid code")
	// ErrInvalidQuantity tells that a quantity is not a power of ten.
	ErrInvalidQuantity = errors.New("rate: invalid quantity")
)
