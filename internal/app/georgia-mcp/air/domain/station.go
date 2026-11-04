package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	minCodeLength = 2
	maxCodeLength = 8
	currentAge    = 24 * time.Hour
)

// StationCode is the code of a station at the NEA: "TSRT".
type StationCode struct {
	code string
}

// ParseStationCode returns the station code of 2 to 8 ASCII letters and digits, in upper
// case.
func ParseStationCode(text string) (StationCode, error) {
	code := strings.ToUpper(strings.TrimSpace(text))
	if len(code) < minCodeLength || len(code) > maxCodeLength ||
		strings.ContainsFunc(code, func(r rune) bool {
			return (r < 'A' || r > 'Z') && (r < '0' || r > '9')
		}) {
		return StationCode{}, fmt.Errorf("%w: %q", ErrInvalidStationCode, text)
	}

	return StationCode{code: code}, nil
}

// String returns the code.
func (c StationCode) String() string {
	return c.code
}

// Name is a text in Georgian and in English.
type Name struct {
	Georgian, English string
}

// In returns the text of the language, or the Georgian text when the English one is empty.
func (n Name) In(language kernel.Language) string {
	if language == kernel.LanguageEnglish && n.English != "" {
		return n.English
	}

	return n.Georgian
}

// Reading is the concentration of one pollutant in one hour.
type Reading struct {
	Value decimal.Decimal
	Time  time.Time
	Level Level
}

// Series is the readings of one pollutant at a station, oldest first.
type Series struct {
	Pollutant Pollutant
	Readings  []Reading
}

// Station is one automatic station of the NEA with its series.
type Station struct {
	Code     StationCode
	CityID   int
	Address  Name
	Lat, Lon float64
	Series   []Series
}

// PollutantReading is a reading with its pollutant.
type PollutantReading struct {
	Pollutant Pollutant
	Reading   Reading
}

// Newest returns the last reading of each series that is less than 24 hours older than
// now. A reading with a time after now counts: the NEA stamps some pollutants ahead of the
// clock.
func (s Station) Newest(now time.Time) []PollutantReading {
	var newest []PollutantReading

	for _, series := range s.Series {
		if len(series.Readings) == 0 {
			continue
		}

		last := series.Readings[len(series.Readings)-1]
		if last.Time.After(now.Add(-currentAge)) {
			newest = append(newest, PollutantReading{Pollutant: series.Pollutant, Reading: last})
		}
	}

	return newest
}

// StationLevel returns the worst level of the readings, or no level.
func StationLevel(readings []PollutantReading) Level {
	var worst Level
	for _, reading := range readings {
		worst = max(worst, reading.Reading.Level)
	}

	return worst
}

// City is a city or a municipality of Georgia.
type City struct {
	ID   int
	Name Name
}
