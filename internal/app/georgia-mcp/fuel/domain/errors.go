package domain

import "errors"

var (
	// ErrInvalidCompanyCode tells that a text is not 2 to 16 lower case letters.
	ErrInvalidCompanyCode = errors.New("company: invalid code")
	// ErrInvalidGrade tells that a text is not one of the grades.
	ErrInvalidGrade = errors.New("grade: invalid")
	// ErrInvalidProductPrice tells that a product lacks a code, a name, a known grade,
	// or a date, or that a price is not more than zero.
	ErrInvalidProductPrice = errors.New("price: invalid product price")
)
