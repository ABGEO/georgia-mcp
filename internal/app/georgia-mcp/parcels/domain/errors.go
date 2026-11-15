package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidCadastralCode tells that a text is not a cadastral code.
	ErrInvalidCadastralCode = errors.New("cadastral code: invalid")
	// ErrPointOutsideGeorgia tells that a point is outside the box of Georgia.
	ErrPointOutsideGeorgia = errors.New("point: outside Georgia")
	// ErrInvalidArea tells that an area has no decimal value or no unit.
	ErrInvalidArea = errors.New("area: invalid")
)

// UnitCodeError tells that a code names a unit under a parcel. It wraps
// ErrInvalidCadastralCode.
type UnitCodeError struct {
	// Parcel is the code of the parcel: the first 5 groups.
	Parcel CadastralCode
}

func (e *UnitCodeError) Error() string {
	return fmt.Sprintf("%v: a unit of the parcel %s", ErrInvalidCadastralCode, e.Parcel)
}

func (e *UnitCodeError) Unwrap() error {
	return ErrInvalidCadastralCode
}
