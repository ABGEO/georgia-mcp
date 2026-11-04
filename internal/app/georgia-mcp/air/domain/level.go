package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Level is an air quality level. The zero Level is no level.
type Level int

// The levels, from the best to the worst.
const (
	LevelGood Level = iota + 1
	LevelFair
	LevelModerate
	LevelPoor
	LevelVeryPoor
)

// String returns the name of the level, or "" for no level.
func (l Level) String() string {
	switch l {
	case LevelGood:
		return "good"
	case LevelFair:
		return "fair"
	case LevelModerate:
		return "moderate"
	case LevelPoor:
		return "poor"
	case LevelVeryPoor:
		return "very_poor"
	default:
		return ""
	}
}

// Bands holds the upper limits of good, fair, moderate, and poor for one pollutant.
type Bands struct {
	upper [4]decimal.Decimal
}

// NewBands returns the bands of four limits above zero, each above the one before.
func NewBands(good, fair, moderate, poor decimal.Decimal) (Bands, error) {
	upper := [4]decimal.Decimal{good, fair, moderate, poor}

	previous := decimal.Zero
	for _, limit := range upper {
		if !limit.GreaterThan(previous) {
			return Bands{}, fmt.Errorf("%w: %s", ErrInvalidBands, upper)
		}

		previous = limit
	}

	return Bands{upper: upper}, nil
}

// Level returns the level whose limits hold the value, the lower limit inclusive. A
// negative value has no level, and the zero Bands give none. A value above the last limit
// is very poor.
func (b Bands) Level(value decimal.Decimal) Level {
	if value.IsNegative() || b == (Bands{}) {
		return 0
	}

	for index, limit := range b.upper {
		if value.LessThan(limit) {
			return LevelGood + Level(index)
		}
	}

	return LevelVeryPoor
}
