package domain

import "errors"

// ErrInvalidHolidayName tells that a holiday name has no text in either language.
var ErrInvalidHolidayName = errors.New("holiday name: invalid")
