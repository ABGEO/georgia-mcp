package domain

import "errors"

var (
	// ErrInvalidTableCode tells that a text is not the path of a table.
	ErrInvalidTableCode = errors.New("table code: invalid")
	// ErrInvalidFigure tells that a text is neither a decimal nor a mark of Geostat for a
	// figure that it did not publish.
	ErrInvalidFigure = errors.New("figure: invalid")
	// ErrInvalidMonth tells that a text is not a month in the form YYYY-MM.
	ErrInvalidMonth = errors.New("month: invalid")
	// ErrUnknownRegion tells that a code is not a region of the consumer price index.
	ErrUnknownRegion = errors.New("region: unknown")
	// ErrUnknownPriceGroup tells that a code is not a price group of the consumer price
	// index.
	ErrUnknownPriceGroup = errors.New("price group: unknown")
)
