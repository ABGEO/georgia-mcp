package domain

import "errors"

var (
	// ErrInvalidText tells that a search text is empty or longer than MaxTextLength.
	ErrInvalidText = errors.New("text: invalid")
	// ErrInvalidID tells that an identifier is not a positive number.
	ErrInvalidID = errors.New("identifier: invalid")
	// ErrInvertedRange tells that a period starts after it ends.
	ErrInvertedRange = errors.New("range: start after end")
)
