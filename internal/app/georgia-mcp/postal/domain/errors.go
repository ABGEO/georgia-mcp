package domain

import "errors"

// ErrInvalidPostalCode tells that a text is not exactly 4 ASCII digits.
var ErrInvalidPostalCode = errors.New("postal code: invalid")
