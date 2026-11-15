package app

import "github.com/abgeo/georgia-mcp/internal/pkg/kernel"

// FirstYear is the first year that the server supports.
const FirstYear = 2025

// SupportedYears returns the first and the last supported year on the date.
func SupportedYears(today kernel.Date) (int, int) {
	return FirstYear, today.Time().Year() + 1
}

func checkYear(year int, today kernel.Date) error {
	first, last := SupportedYears(today)
	if year < first || year > last {
		return &YearError{Year: year, First: first, Last: last}
	}

	return nil
}
