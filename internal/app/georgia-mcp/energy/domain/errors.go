package domain

import "errors"

// ErrInvalidMoment tells that a text is not a moment.
var ErrInvalidMoment = errors.New("moment: invalid")

// ErrInvalidMonth tells that a text is not a month.
var ErrInvalidMonth = errors.New("month: invalid")
