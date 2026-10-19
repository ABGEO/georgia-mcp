package domain

import (
	"fmt"
	"time"
)

// Timing tells how a trip plan uses its time.
type Timing string

const (
	// TimingLeaveNow plans a trip that leaves at the instant of the request.
	TimingLeaveNow Timing = "leave_now"
	// TimingDepartAt plans a trip that leaves at a time.
	TimingDepartAt Timing = "depart_at"
	// TimingArriveBy plans a trip that arrives by a time.
	TimingArriveBy Timing = "arrive_by"
)

// ParseTiming returns the timing of a text. The empty text gives TimingLeaveNow.
func ParseTiming(text string) (Timing, error) {
	switch timing := Timing(text); timing {
	case "":
		return TimingLeaveNow, nil
	case TimingLeaveNow, TimingDepartAt, TimingArriveBy:
		return timing, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidTiming, text)
	}
}

// Preference is what a trip plan makes as small as it can.
type Preference string

const (
	// PreferenceFastest plans the trip of the shortest time.
	PreferenceFastest Preference = "fastest"
	// PreferenceLessWalking plans the trip of the least walk.
	PreferenceLessWalking Preference = "less_walking"
	// PreferenceFewerTransfers plans the trip of the fewest transfers.
	PreferenceFewerTransfers Preference = "fewer_transfers"
)

// ParsePreference returns the preference of a text. The empty text gives
// PreferenceFastest.
func ParsePreference(text string) (Preference, error) {
	switch preference := Preference(text); preference {
	case "":
		return PreferenceFastest, nil
	case PreferenceFastest, PreferenceLessWalking, PreferenceFewerTransfers:
		return preference, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidPreference, text)
	}
}

// LegEnd is where a leg starts or ends.
type LegEnd struct {
	Name string
	// Stop is empty when the end is not a stop.
	Stop     StopID
	Location Point
}

// Leg is one part of an itinerary: a walk, or a ride on one route.
type Leg struct {
	// Kind is KindWalk, or the kind of the route of a ride.
	Kind   Kind
	From   LegEnd
	To     LegEnd
	Start  time.Time
	End    time.Time
	Metres float64
	// RouteNumber and RouteName are empty for a walk.
	RouteNumber string
	RouteName   string
	// Stops counts the stops of a ride after the stop where the person boards.
	Stops int
}

// Itinerary is one way to travel between two points.
type Itinerary struct {
	Start      time.Time
	End        time.Time
	WalkMetres float64
	Legs       []Leg
}

// Duration returns the time from the start to the end.
func (i Itinerary) Duration() time.Duration {
	return i.End.Sub(i.Start)
}

// Transfers returns the count of changes from one vehicle to another.
func (i Itinerary) Transfers() int {
	rides := 0

	for _, leg := range i.Legs {
		if leg.Kind != KindWalk {
			rides++
		}
	}

	return max(rides-1, 0)
}
