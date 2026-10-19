package domain

import (
	"fmt"
	"slices"
)

// Kind is the type of vehicle of a route, or the type of a stop.
type Kind string

const (
	// KindBus is a bus of the TTC, or a bus stop.
	KindBus Kind = "bus"
	// KindMinibus is a minibus. It stops at a bus stop.
	KindMinibus Kind = "minibus"
	// KindMetro is the metro, or a metro station.
	KindMetro Kind = "metro"
	// KindCableCar is a cable car, or a cable car station.
	KindCableCar Kind = "cable_car"
	// KindWalk is a walk of a trip. No constructor accepts it.
	KindWalk Kind = "walk"
)

// StopKinds returns the kinds of a stop.
func StopKinds() []Kind {
	return []Kind{KindBus, KindMetro, KindCableCar}
}

// RouteKinds returns the kinds of a route.
func RouteKinds() []Kind {
	return []Kind{KindBus, KindMinibus, KindMetro, KindCableCar}
}

// ParseStopKind returns the kind of stop of a text. It accepts the exact value only.
func ParseStopKind(text string) (Kind, error) {
	return parseKind(text, StopKinds())
}

// ParseRouteKind returns the kind of route of a text. It accepts the exact value only.
func ParseRouteKind(text string) (Kind, error) {
	return parseKind(text, RouteKinds())
}

// ParseTripKinds returns the kinds that a trip can ride, once each, in the order of
// the texts. No text gives every kind of route. The TTC plans the bus and the minibus
// as one mode, so a trip takes both or neither.
func ParseTripKinds(texts []string) ([]Kind, error) {
	if len(texts) == 0 {
		return RouteKinds(), nil
	}

	kinds := make([]Kind, 0, len(texts))

	for _, text := range texts {
		kind, err := ParseRouteKind(text)
		if err != nil {
			return nil, err
		}

		if !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}

	if slices.Contains(kinds, KindBus) != slices.Contains(kinds, KindMinibus) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTripKinds, texts)
	}

	return kinds, nil
}

// StopKind returns the kind of the stop where a vehicle of the kind stops.
func (k Kind) StopKind() Kind {
	if k == KindMinibus {
		return KindBus
	}

	return k
}

func parseKind(text string, kinds []Kind) (Kind, error) {
	kind := Kind(text)
	if !slices.Contains(kinds, kind) {
		return "", fmt.Errorf("%w: %q", ErrInvalidKind, text)
	}

	return kind, nil
}
