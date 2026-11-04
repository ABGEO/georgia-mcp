package domain

import (
	"fmt"
	"slices"
	"strings"
)

// Pollutant is one substance that a station measures.
type Pollutant string

// The pollutants of the NEA.
const (
	PM10 Pollutant = "PM10"
	PM25 Pollutant = "PM2.5"
	NO2  Pollutant = "NO2"
	O3   Pollutant = "O3"
	SO2  Pollutant = "SO2"
	CO   Pollutant = "CO"
)

// Pollutants returns every pollutant, in a fixed order.
func Pollutants() []Pollutant {
	return []Pollutant{PM10, PM25, NO2, O3, SO2, CO}
}

// ParsePollutant returns the pollutant of a name, with no regard to case.
func ParsePollutant(text string) (Pollutant, error) {
	index := slices.IndexFunc(Pollutants(), func(p Pollutant) bool {
		return strings.EqualFold(string(p), strings.TrimSpace(text))
	})
	if index < 0 {
		return "", fmt.Errorf("%w: %q", ErrUnknownPollutant, text)
	}

	return Pollutants()[index], nil
}

// Unit returns the unit of the concentration of the pollutant.
func (p Pollutant) Unit() string {
	if p == CO {
		return "mg/m³"
	}

	return "µg/m³"
}
