package domain_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	bus      = "bus"
	minibus  = "minibus"
	metro    = "metro"
	cableCar = "cable_car"
	walk     = "walk"
	morning  = "06:45"
	midnight = "24:05"
	vakePark = "ვაკის პარკი"
)

// TTCTRANS-SC-009: A kind of stop is bus, metro, or cable_car. A kind of route adds
// minibus. Any other text is refused.
func TestParseKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input        string
		stopErr      error
		routeErr     error
		wantStopKind domain.Kind
	}{
		{input: bus, wantStopKind: domain.KindBus},
		{input: metro, wantStopKind: domain.KindMetro},
		{input: cableCar, wantStopKind: domain.KindCableCar},
		{input: minibus, stopErr: domain.ErrInvalidKind, wantStopKind: domain.KindBus},
		{input: "BUS", stopErr: domain.ErrInvalidKind, routeErr: domain.ErrInvalidKind},
		{input: walk, stopErr: domain.ErrInvalidKind, routeErr: domain.ErrInvalidKind},
		{input: "", stopErr: domain.ErrInvalidKind, routeErr: domain.ErrInvalidKind},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.ParseStopKind(tt.input); !errors.Is(err, tt.stopErr) {
				t.Errorf("ParseStopKind(%q) error = %v, want %v", tt.input, err, tt.stopErr)
			}

			kind, err := domain.ParseRouteKind(tt.input)
			if !errors.Is(err, tt.routeErr) {
				t.Errorf("ParseRouteKind(%q) error = %v, want %v", tt.input, err, tt.routeErr)
			}

			if err == nil && kind.StopKind() != tt.wantStopKind {
				t.Errorf("%q.StopKind() = %q, want %q", kind, kind.StopKind(), tt.wantStopKind)
			}
		})
	}
}

// TTCTRANS-SC-009: No input panics, and every accepted kind is a kind of the list.
func FuzzParseRouteKind(f *testing.F) {
	for _, seed := range []string{bus, minibus, "BUS", walk, "", "ავტობუსი"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		kind, err := domain.ParseRouteKind(input)
		if err != nil {
			if !errors.Is(err, domain.ErrInvalidKind) {
				t.Fatalf("ParseRouteKind(%q) error = %v, want ErrInvalidKind", input, err)
			}

			return
		}

		if string(kind) != input {
			t.Fatalf("ParseRouteKind(%q) = %q", input, kind)
		}
	})
}

// TTCTRANS-SC-010: A trip takes bus and minibus together, or neither. No kind asks
// every kind. A kind named twice counts once.
func TestParseTripKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []string
		want    []domain.Kind
		wantErr error
	}{
		{name: "none", want: domain.RouteKinds()},
		{
			name:  "bus, minibus, metro",
			input: []string{bus, minibus, metro},
			want:  []domain.Kind{domain.KindBus, domain.KindMinibus, domain.KindMetro},
		},
		{name: "minibus alone", input: []string{minibus}, wantErr: domain.ErrInvalidTripKinds},
		{name: "bus and metro", input: []string{bus, metro}, wantErr: domain.ErrInvalidTripKinds},
		{
			name:  "bus twice",
			input: []string{bus, bus, minibus},
			want:  []domain.Kind{domain.KindBus, domain.KindMinibus},
		},
		{name: walk, input: []string{walk}, wantErr: domain.ErrInvalidKind},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseTripKinds(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseTripKinds(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ParseTripKinds(%q) mismatch (-want +got):\n%s", tt.input, diff)
			}
		})
	}
}

// TTCTRANS-SC-011: A point needs a latitude and a longitude in range. Tbilisi is a box.
// The distance from a point to the stop 1:1000 rounds to 3 metres.
func TestPoint(t *testing.T) {
	t.Parallel()

	for _, bad := range [][2]float64{{95, 44.8}, {41.7, 181}, {math.NaN(), 44.8}} {
		if _, err := domain.NewPoint(bad[0], bad[1]); !errors.Is(err, domain.ErrInvalidPoint) {
			t.Errorf("NewPoint(%v, %v) error = %v, want ErrInvalidPoint", bad[0], bad[1], err)
		}
	}

	far, err := domain.NewPoint(40, 40)
	if err != nil {
		t.Fatal(err)
	}

	if far.InTbilisi() {
		t.Error("40, 40 is in Tbilisi, want outside")
	}

	from, err := domain.NewPoint(41.7251, 44.7554)
	if err != nil {
		t.Fatal(err)
	}

	stop, err := domain.NewPoint(41.725095, 44.755362)
	if err != nil {
		t.Fatal(err)
	}

	if !from.InTbilisi() {
		t.Error("41.7251, 44.7554 is outside Tbilisi, want inside")
	}

	if got := math.Round(from.MetresTo(stop)); got != 3 {
		t.Errorf("MetresTo() = %v, want 3 after the round", got)
	}
}

// TTCTRANS-SC-012: A clock time is H:MM or HH:MM, with an hour of 0 to 47.
func TestParseClockTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{input: "6:45", want: morning},
		{input: morning, want: morning},
		{input: midnight, want: midnight},
		{input: "47:59", want: "47:59"},
		{input: "48:00", wantErr: domain.ErrInvalidClockTime},
		{input: "6:60", wantErr: domain.ErrInvalidClockTime},
		{input: "645", wantErr: domain.ErrInvalidClockTime},
		{input: "6:5", wantErr: domain.ErrInvalidClockTime},
		{input: "", wantErr: domain.ErrInvalidClockTime},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseClockTime(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseClockTime(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}

			if err == nil && got.String() != tt.want {
				t.Errorf("ParseClockTime(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

// TTCTRANS-SC-012: No input panics, and each accepted time formats to HH:MM and parses
// back to the same time.
func FuzzParseClockTime(f *testing.F) {
	for _, seed := range []string{"6:45", midnight, "48:00", "6:60", "-1:00", "+6:45", "٦:٤٥"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		clock, err := domain.ParseClockTime(input)
		if err != nil {
			return
		}

		text := clock.String()
		if len(text) != len("HH:MM") {
			t.Fatalf("ParseClockTime(%q).String() = %q, want HH:MM", input, text)
		}

		again, err := domain.ParseClockTime(text)
		if err != nil || again != clock {
			t.Fatalf("ParseClockTime(%q) = %v, %v, want %v", text, again, err, clock)
		}
	})
}

// TTCTRANS-SC-013: A name gives the requested language, or the other one when the
// requested one is empty.
func TestNameIn(t *testing.T) {
	t.Parallel()

	georgianOnly := domain.Name{Georgian: vakePark}
	if got := georgianOnly.In(kernel.LanguageEnglish); got != vakePark {
		t.Errorf("In(en) = %q, want the Georgian name", got)
	}

	both := domain.Name{Georgian: vakePark, English: "Vake Park"}
	if got := both.In(kernel.LanguageEnglish); got != "Vake Park" {
		t.Errorf("In(en) = %q, want Vake Park", got)
	}

	if got := both.In(kernel.LanguageGeorgian); got != vakePark {
		t.Errorf("In(ka) = %q, want ვაკის პარკი", got)
	}
}

// TTCTRANS-SC-014: The transfers of an itinerary are its rides minus one. The duration
// is the end minus the start.
func TestItinerary(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 19, 21, 42, 48, 0, kernel.TbilisiZone())
	walk := domain.Leg{Kind: domain.KindWalk}
	ride := domain.Leg{Kind: domain.KindBus}

	trip := domain.Itinerary{
		Start: start,
		End:   start.Add(61 * time.Minute),
		Legs:  []domain.Leg{walk, ride, ride, walk},
	}

	if got := trip.Transfers(); got != 1 {
		t.Errorf("Transfers() = %d, want 1", got)
	}

	if got := trip.Duration(); got != 61*time.Minute {
		t.Errorf("Duration() = %v, want 61m", got)
	}

	if got := (domain.Itinerary{Legs: []domain.Leg{walk}}).Transfers(); got != 0 {
		t.Errorf("Transfers() of a walk = %d, want 0", got)
	}
}
