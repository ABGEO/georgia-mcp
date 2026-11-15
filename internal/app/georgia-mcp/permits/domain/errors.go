package domain

import "errors"

var (
	// ErrInvalidCadastralCode tells that a text is not a cadastral code.
	ErrInvalidCadastralCode = errors.New("cadastral code: invalid")
	// ErrInvalidApplicationNumber tells that a text is not an application number.
	ErrInvalidApplicationNumber = errors.New("application number: invalid")
	// ErrInvalidPeriod tells that a period starts after it ends.
	ErrInvalidPeriod = errors.New("period: the start is after the end")
	// ErrPeriodTooLong tells that a period holds more than MaxPeriodDays days.
	ErrPeriodTooLong = errors.New("period: longer than 31 days")
)
