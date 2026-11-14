package domain_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const station = "TSRT"

// NEAAIR-SC-004: A pollutant parses with no regard to case.
func TestParsePollutant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    domain.Pollutant
		wantErr error
	}{
		{input: "PM2.5", want: domain.PM25},
		{input: "pm2.5", want: domain.PM25},
		{input: "co", want: domain.CO},
		{input: " NO2 ", want: domain.NO2},
		{input: "PM1", wantErr: domain.ErrUnknownPollutant},
		{input: "PM25", wantErr: domain.ErrUnknownPollutant},
		{input: "", wantErr: domain.ErrUnknownPollutant},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParsePollutant(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParsePollutant(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("ParsePollutant(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// NEAAIR-SC-004: The pollutants come in a fixed order, CO in mg/m³.
func TestPollutantUnits(t *testing.T) {
	t.Parallel()

	want := []domain.Pollutant{
		domain.PM10, domain.PM25, domain.NO2, domain.O3, domain.SO2, domain.CO,
	}
	if diff := cmp.Diff(want, domain.Pollutants()); diff != "" {
		t.Errorf("Pollutants() mismatch (-want +got):\n%s", diff)
	}

	if domain.CO.Unit() != "mg/m³" || domain.PM10.Unit() != "µg/m³" {
		t.Errorf("units = %q, %q", domain.CO.Unit(), domain.PM10.Unit())
	}
}

// NEAAIR-SC-004: Each pollutant that parses is one of Pollutants().
func FuzzParsePollutant(f *testing.F) {
	for _, seed := range []string{"PM2.5", "pm10", "PM1", "", "ＣＯ"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got, err := domain.ParsePollutant(input)
		if err == nil && !slices.Contains(domain.Pollutants(), got) {
			t.Errorf("ParsePollutant(%q) = %q, not a pollutant", input, got)
		}
	})
}

// NEAAIR-SC-005: A station code is 2 to 8 ASCII letters and digits, in upper case.
func TestParseStationCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{input: "tsrt", want: station},
		{input: station, want: station},
		{input: "01005", want: "01005"},
		{input: "X", wantErr: domain.ErrInvalidStationCode},
		{input: "TOOLONGCODE", wantErr: domain.ErrInvalidStationCode},
		{input: "TS-RT", wantErr: domain.ErrInvalidStationCode},
		{input: "ТSRT", wantErr: domain.ErrInvalidStationCode},
		{input: "ſA", wantErr: domain.ErrInvalidStationCode},
		{input: "", wantErr: domain.ErrInvalidStationCode},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseStationCode(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseStationCode(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if got.String() != tt.want {
				t.Errorf("ParseStationCode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// NEAAIR-SC-005: Each station code that parses is 2 to 8 upper case letters and digits.
func FuzzParseStationCode(f *testing.F) {
	for _, seed := range []string{station, "tsrt", "01005", "X", "ТSRT", ""} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		code, err := domain.ParseStationCode(input)
		if err != nil {
			return
		}

		text := code.String()
		if len(text) < 2 || len(text) > 8 {
			t.Errorf("ParseStationCode(%q) = %q, wrong length", input, text)
		}

		for _, r := range text {
			if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				t.Errorf("ParseStationCode(%q) = %q, holds %q", input, text, r)
			}
		}
	})
}

func number(text string) decimal.Decimal {
	return decimal.RequireFromString(text)
}

func pm10Bands(t *testing.T) domain.Bands {
	t.Helper()

	bands, err := domain.NewBands(number("20"), number("35"), number("50"), number("100"))
	if err != nil {
		t.Fatalf("NewBands() error = %v", err)
	}

	return bands
}

// NEAAIR-SC-006: A value takes the level whose limits hold it, the lower limit inclusive.
func TestBandsLevel(t *testing.T) {
	t.Parallel()

	bands := pm10Bands(t)

	tests := []struct {
		value string
		want  domain.Level
	}{
		{value: "19.99", want: domain.LevelGood},
		{value: "0", want: domain.LevelGood},
		{value: "20", want: domain.LevelFair},
		{value: "35", want: domain.LevelModerate},
		{value: "50", want: domain.LevelPoor},
		{value: "51.8833333333333", want: domain.LevelPoor},
		{value: "100", want: domain.LevelVeryPoor},
		{value: "1300", want: domain.LevelVeryPoor},
		{value: "-0.8"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			if got := bands.Level(number(tt.value)); got != tt.want {
				t.Errorf("Level(%s) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// NEAAIR-SC-006: The zero bands give no level.
func TestZeroBandsGiveNoLevel(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"0", "19.99", "1300"} {
		if got := (domain.Bands{}).Level(number(value)); got != 0 {
			t.Errorf("Bands{}.Level(%s) = %v, want no level", value, got)
		}
	}
}

// NEAAIR-SC-006: The limits must rise above zero.
func TestNewBandsRefusesBadLimits(t *testing.T) {
	t.Parallel()

	tests := [][4]string{
		{"20", "10", "50", "100"},
		{"0", "35", "50", "100"},
		{"20", "20", "50", "100"},
	}

	for _, limits := range tests {
		_, err := domain.NewBands(
			number(limits[0]), number(limits[1]), number(limits[2]), number(limits[3]),
		)
		if !errors.Is(err, domain.ErrInvalidBands) {
			t.Errorf("NewBands(%v) error = %v, want ErrInvalidBands", limits, err)
		}
	}
}

// NEAAIR-SC-006: The level names, and no name for no level.
func TestLevelString(t *testing.T) {
	t.Parallel()

	got := make([]string, 0, 6)
	for _, level := range []domain.Level{
		0, domain.LevelGood, domain.LevelFair, domain.LevelModerate, domain.LevelPoor,
		domain.LevelVeryPoor,
	} {
		got = append(got, level.String())
	}

	want := []string{"", "good", "fair", "moderate", "poor", "very_poor"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("String() mismatch (-want +got):\n%s", diff)
	}
}

// NEAAIR-SC-006: The level never falls when the value grows.
func FuzzBandsLevel(f *testing.F) {
	f.Add(int64(0), int64(1))
	f.Add(int64(-8), int64(1300))
	f.Add(int64(199), int64(200))

	bands, err := domain.NewBands(number("20"), number("35"), number("50"), number("100"))
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, low, high int64) {
		if low > high {
			low, high = high, low
		}

		lowValue, highValue := decimal.New(low, -1), decimal.New(high, -1)
		if bands.Level(lowValue) > bands.Level(highValue) && !lowValue.IsNegative() {
			t.Errorf("Level(%s) > Level(%s)", lowValue, highValue)
		}
	})
}

func at(t *testing.T, text string) time.Time {
	t.Helper()

	instant, err := time.ParseInLocation("2006-01-02T15:04", text, kernel.TbilisiZone())
	if err != nil {
		t.Fatal(err)
	}

	return instant
}

func reading(t *testing.T, value, hour string, level domain.Level) domain.Reading {
	t.Helper()

	return domain.Reading{Value: number(value), Time: at(t, hour), Level: level}
}

// NEAAIR-SC-007: The newest reading of each series counts when it is less than 24 hours
// old, a time ahead of now too. The station level is the worst of them.
func TestStationNewest(t *testing.T) {
	t.Parallel()

	tsrt := domain.Station{Series: []domain.Series{
		{Pollutant: domain.PM10, Readings: []domain.Reading{
			reading(t, "50.795", "2026-11-05T03:00", domain.LevelPoor),
			reading(t, "51.8833333333333", "2026-11-05T05:00", domain.LevelPoor),
		}},
		{Pollutant: domain.NO2, Readings: []domain.Reading{
			reading(t, "46.55", "2026-11-04T23:00", domain.LevelFair),
		}},
		{Pollutant: domain.SO2, Readings: []domain.Reading{
			reading(t, "6.6", "2026-11-04T23:00", domain.LevelGood),
		}},
		{Pollutant: domain.CO},
	}}

	now := at(t, "2026-11-05T00:28")

	newest := tsrt.Newest(now)

	want := []domain.PollutantReading{
		{Pollutant: domain.PM10, Reading: tsrt.Series[0].Readings[1]},
		{Pollutant: domain.NO2, Reading: tsrt.Series[1].Readings[0]},
		{Pollutant: domain.SO2, Reading: tsrt.Series[2].Readings[0]},
	}
	if diff := cmp.Diff(want, newest); diff != "" {
		t.Errorf("Newest() mismatch (-want +got):\n%s", diff)
	}

	if got := domain.StationLevel(newest); got != domain.LevelPoor {
		t.Errorf("StationLevel() = %v, want poor", got)
	}

	agms := domain.Station{Series: []domain.Series{
		{Pollutant: domain.PM10, Readings: []domain.Reading{
			reading(t, "52.5354030277778", "2026-11-04T19:00", domain.LevelPoor),
		}},
	}}

	if got := agms.Newest(now); len(got) != 1 {
		t.Errorf("AGMS at now: %d readings, want 1", len(got))
	}

	later := agms.Newest(now.Add(24 * time.Hour))
	if len(later) != 0 || domain.StationLevel(later) != 0 {
		t.Errorf("AGMS at now plus 24 hours: %v, want no reading and no level", later)
	}
}

// NEAAIR-SC-007: A reading with no level gives no station level alone.
func TestStationLevelIgnoresNoLevel(t *testing.T) {
	t.Parallel()

	readings := []domain.PollutantReading{
		{Pollutant: domain.CO, Reading: reading(t, "-0.8", "2026-11-05T01:00", 0)},
	}
	if got := domain.StationLevel(readings); got != 0 {
		t.Errorf("StationLevel() = %v, want no level", got)
	}
}

// NEAAIR-SC-009: A name falls back to Georgian when English is empty.
func TestNameIn(t *testing.T) {
	t.Parallel()

	name := domain.Name{Georgian: "თბილისი", English: "Tbilisi"}
	if name.In(kernel.LanguageEnglish) != "Tbilisi" ||
		name.In(kernel.LanguageGeorgian) != "თბილისი" {
		t.Errorf("In() = %q, %q", name.In(kernel.LanguageEnglish), name.In(kernel.LanguageGeorgian))
	}

	georgian := domain.Name{Georgian: "მესტია"}
	if georgian.In(kernel.LanguageEnglish) != "მესტია" {
		t.Errorf("In(en) = %q, want the Georgian name", georgian.In(kernel.LanguageEnglish))
	}
}
