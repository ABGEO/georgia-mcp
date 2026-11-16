package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func tbilisi(hour, minute, second, nanosecond int) time.Time {
	return time.Date(
		2026,
		time.November,
		16,
		hour,
		minute,
		second,
		nanosecond,
		kernel.TbilisiZone(),
	)
}

// GSEGRID-SC-004: A moment with an offset keeps it. A moment with no offset is a time of
// Tbilisi.
func TestParseMoment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    time.Time
		wantErr bool
	}{
		{input: "2026-11-16T13:01", want: tbilisi(13, 1, 0, 0)},
		{input: "2026-11-16 13:01", want: tbilisi(13, 1, 0, 0)},
		{input: "2026-11-16T13:01:00", want: tbilisi(13, 1, 0, 0)},
		{input: " 2026-11-16 13:01:30 ", want: tbilisi(13, 1, 30, 0)},
		{input: "2026-11-16T09:01:00Z", want: tbilisi(13, 1, 0, 0)},
		{input: "2026-11-16T13:01:00+04:00", want: tbilisi(13, 1, 0, 0)},
		{input: "2026-11-16T09:01Z", want: tbilisi(13, 1, 0, 0)},
		{input: "yesterday", wantErr: true},
		{input: "2026-11-16", wantErr: true},
		{input: "13:01", wantErr: true},
		{input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseMoment(tt.input)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidMoment) {
					t.Errorf("ParseMoment(%q) error = %v, want ErrInvalidMoment", tt.input, err)
				}

				return
			}

			if err != nil || !got.Equal(tt.want) {
				t.Errorf("ParseMoment(%q) = %v, %v, want %v", tt.input, got, err, tt.want)
			}
		})
	}
}

// GSEGRID-SC-004: A moment parses back from its RFC 3339 text.
func FuzzParseMoment(f *testing.F) {
	for _, seed := range []string{
		"2026-11-16T13:01", "2026-11-16 13:01:30", "2026-11-16T09:01:00Z",
		"2026-11-16T13:01:00.5+04:00", "yesterday", "",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got, err := domain.ParseMoment(input)
		if err != nil {
			return
		}

		back, err := time.Parse(time.RFC3339Nano, got.Format(time.RFC3339Nano))
		if err != nil || !back.Equal(got) {
			t.Errorf("ParseMoment(%q) = %v, which does not parse back: %v", input, got, err)
		}
	})
}

// GSEGRID-SC-005: A slot starts at a multiple of 3 minutes, in Tbilisi.
func TestSlotOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Time
		want  time.Time
	}{
		{name: "13:01", input: tbilisi(13, 1, 0, 0), want: tbilisi(13, 0, 0, 0)},
		{
			name:  "09:01Z",
			input: time.Date(2026, time.November, 16, 9, 1, 0, 0, time.UTC),
			want:  tbilisi(13, 0, 0, 0),
		},
		{name: "13:00:00", input: tbilisi(13, 0, 0, 0), want: tbilisi(13, 0, 0, 0)},
		{
			name:  "12:59:59.999",
			input: tbilisi(12, 59, 59, 999_000_000),
			want:  tbilisi(12, 57, 0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := domain.SlotOf(tt.input)
			if !got.Equal(tt.want) || got.Location().String() != kernel.TbilisiZone().String() {
				t.Errorf("SlotOf(%v) = %v, want %v in Tbilisi", tt.input, got, tt.want)
			}
		})
	}
}

// GSEGRID-SC-006: Each quality code has its meaning. Only ACT is measured.
func TestQuality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code     domain.Quality
		meaning  string
		measured bool
	}{
		{code: domain.QualityMeasured, meaning: "measured", measured: true},
		{code: domain.QualitySubstituted, meaning: "substituted by the operator"},
		{code: domain.QualityNotReceived, meaning: "not received"},
		{code: domain.QualityBLS, meaning: "not documented by GSE"},
		{code: "XYZ", meaning: "unknown"},
		{code: "", meaning: "unknown"},
	}

	for _, tt := range tests {
		if got := tt.code.Meaning(); got != tt.meaning {
			t.Errorf("Quality(%q).Meaning() = %q, want %q", tt.code, got, tt.meaning)
		}

		if got := tt.code.Measured(); got != tt.measured {
			t.Errorf("Quality(%q).Measured() = %v, want %v", tt.code, got, tt.measured)
		}
	}
}

// GSEGRID-SC-007: A positive number is an import, a negative one an export, a zero none.
func TestDirectionOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		number string
		want   domain.Direction
	}{
		{number: "279.235687", want: domain.DirectionImport},
		{number: "-23.05", want: domain.DirectionExport},
		{number: "0.000000", want: domain.DirectionNone},
		{number: "-0.000000", want: domain.DirectionNone},
	}

	for _, tt := range tests {
		if got := domain.DirectionOf(decimal.RequireFromString(tt.number)); got != tt.want {
			t.Errorf("DirectionOf(%s) = %q, want %q", tt.number, got, tt.want)
		}
	}
}

func value(number string, quality domain.Quality) domain.Value {
	return domain.Value{Number: decimal.RequireFromString(number), Quality: quality}
}

func line(name, number string, quality domain.Quality) domain.Line {
	return domain.Line{Name: name, Value: value(number, quality)}
}

// recordedState is the grid state of the file of 2026-11-16 12:00.
func recordedState() domain.GridState {
	return domain.GridState{
		Slot: tbilisi(12, 0, 0, 0),
		Generation: domain.Generation{
			Hydro:   value("1331.362305", domain.QualitySubstituted),
			Thermal: value("170.777710", domain.QualitySubstituted),
			Wind:    value("-0.036000", domain.QualityNotReceived),
			Solar:   value("6.000000", domain.QualitySubstituted),
			Total:   value("1511.088379", domain.QualityNotReceived),
		},
		Consumption: value("1843.806030", domain.QualityNotReceived),
		Flows: []domain.Flow{
			{
				Country: "Azerbaijan",
				Lines:   []domain.Line{line("Azerbaij", "20.631090", domain.QualityMeasured)},
			},
			{
				Country: "Armenia",
				Lines:   []domain.Line{line("Armenia", "0.000000", domain.QualitySubstituted)},
			},
			{Country: "Russia", Lines: []domain.Line{
				line("Russia", "0.000000", domain.QualityMeasured),
				line("Salkhino", "-0.000000", domain.QualityMeasured),
				line("Java", "18.299999", domain.QualityBLS),
				line("Nakaduli", "14.191387", domain.QualityMeasured),
			}},
			{
				Country: "Turkey",
				Lines:   []domain.Line{line("Turkey", "279.235687", domain.QualityMeasured)},
			},
		},
	}
}

// GSEGRID-SC-008: The sums keep the exact decimals, and the state counts its measured
// values.
func TestGridStateSums(t *testing.T) {
	t.Parallel()

	state := recordedState()

	russia := state.Flows[2].Net()
	if russia.String() != "32.491386" || russia.Exponent() != -6 {
		t.Errorf("Net() of Russia = %s with the exponent %d, want 32.491386 with -6",
			russia, russia.Exponent())
	}

	if got := state.NetImport(); got.String() != "332.358163" {
		t.Errorf("NetImport() = %s, want 332.358163", got)
	}

	if got := len(state.Values()); got != 13 {
		t.Errorf("len(Values()) = %d, want 13", got)
	}

	if got := state.Measured(); got != 5 {
		t.Errorf("Measured() = %d, want 5", got)
	}
}
