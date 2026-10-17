package kernel

import "errors"

var (
	// ErrSourceUnavailable tells that a vendor did not answer. A driven adapter
	// wraps every failure of a vendor with it.
	ErrSourceUnavailable = errors.New("source: unavailable")

	// ErrUnknownLanguage tells that a text is not a known language code.
	ErrUnknownLanguage = errors.New("language: unknown code")
)
