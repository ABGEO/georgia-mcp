package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// stopMatchMetres is the distance within which an end of a leg is the stop. The TTC
// gives a leg end the exact coordinates of its stop.
const stopMatchMetres = 2

// TripQuery asks the itineraries between two points.
type TripQuery struct {
	From domain.Point
	To   domain.Point
	// Timing is empty for TimingLeaveNow.
	Timing domain.Timing
	// Date is zero for the date in Tbilisi. Time is required unless the timing is
	// TimingLeaveNow.
	Date kernel.Date
	Time *domain.ClockTime
	// Kinds is empty for every kind.
	Kinds []domain.Kind
	// Preference is empty for PreferenceFastest.
	Preference domain.Preference
	Language   kernel.Language
}

// Trip is the itineraries of a trip plan.
type Trip struct {
	Itineraries []domain.Itinerary
	ReadAt      time.Time
}

// PlanTrip plans a trip between two points of Tbilisi.
type PlanTrip struct {
	network NetworkSource
	planner TripPlanner
	now     func() time.Time
}

// NewPlanTrip returns the use case. now gives the date of a query with a time and no
// date.
func NewPlanTrip(network NetworkSource, planner TripPlanner, now func() time.Time) *PlanTrip {
	return &PlanTrip{network: network, planner: planner, now: now}
}

// Plan returns the itineraries. Each end of a leg at a stop carries the stop, and each
// ride carries the kind of its route.
func (u *PlanTrip) Plan(ctx context.Context, query TripQuery) (Trip, error) {
	request, err := u.request(query)
	if err != nil {
		return Trip{}, err
	}

	stops, err := u.network.Stops(ctx)
	if err != nil {
		return Trip{}, fmt.Errorf("reading the stops: %w", err)
	}

	routes, err := u.network.Routes(ctx)
	if err != nil {
		return Trip{}, fmt.Errorf("reading the routes: %w", err)
	}

	plan, err := u.planner.Plan(ctx, request)
	if err != nil {
		return Trip{}, fmt.Errorf("planning the trip: %w", err)
	}

	itineraries := make([]domain.Itinerary, 0, len(plan.Itineraries))

	for _, itinerary := range plan.Itineraries {
		itinerary.Legs = slices.Clone(itinerary.Legs)

		for i := range itinerary.Legs {
			leg := &itinerary.Legs[i]
			leg.From.Stop = stopAt(stops.Stops, leg.From.Location)
			leg.To.Stop = stopAt(stops.Stops, leg.To.Location)

			if leg.Kind != domain.KindWalk {
				leg.Kind = classify(routes.Routes, leg.RouteNumber, leg.Kind.StopKind())
			}
		}

		itineraries = append(itineraries, itinerary)
	}

	return Trip{Itineraries: itineraries, ReadAt: plan.ReadAt}, nil
}

func (u *PlanTrip) request(query TripQuery) (TripRequest, error) {
	if !query.From.InTbilisi() {
		return TripRequest{}, &OutsideTbilisiError{Field: "from", Point: query.From}
	}

	if !query.To.InTbilisi() {
		return TripRequest{}, &OutsideTbilisiError{Field: "to", Point: query.To}
	}

	request := TripRequest{
		From:       query.From,
		To:         query.To,
		Timing:     query.Timing,
		Kinds:      query.Kinds,
		Preference: query.Preference,
		Language:   query.Language,
	}

	if request.Timing == "" {
		request.Timing = domain.TimingLeaveNow
	}

	if len(request.Kinds) == 0 {
		request.Kinds = domain.RouteKinds()
	}

	if request.Preference == "" {
		request.Preference = domain.PreferenceFastest
	}

	if request.Timing == domain.TimingLeaveNow {
		return request, nil
	}

	if query.Time == nil {
		return TripRequest{}, fmt.Errorf(
			"%w: %s needs a time",
			domain.ErrInvalidClockTime,
			request.Timing,
		)
	}

	date := query.Date
	if date.IsZero() {
		date = kernel.Today(u.now())
	}

	midnight := date.Time()
	request.At = time.Date(
		midnight.Year(), midnight.Month(), midnight.Day(), 0, 0, 0, 0, kernel.TbilisiZone(),
	).Add(time.Duration(query.Time.Minutes()) * time.Minute)

	return request, nil
}

// stopAt returns the stop at a point, or the empty identifier when no stop is there.
func stopAt(stops []domain.Stop, point domain.Point) domain.StopID {
	for _, stop := range stops {
		if stop.Location.MetresTo(point) <= stopMatchMetres {
			return stop.ID
		}
	}

	return ""
}
