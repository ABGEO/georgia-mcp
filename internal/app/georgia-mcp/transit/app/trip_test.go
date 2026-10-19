package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func origin() domain.Point { return domain.Point{Lat: 41.7251, Lon: 44.7554} }

func destination() domain.Point { return domain.Point{Lat: 41.7151, Lon: 44.8271} }

func at(text string) time.Time {
	return fixedNow(text)()
}

// fixtureItinerary is the first itinerary of plan.json, as the adapter converts it: no
// stop identifier, and the kind bus for each ride of the mode BUS.
func fixtureItinerary() domain.Itinerary {
	end := func(name string, lat, lon float64) domain.LegEnd {
		return domain.LegEnd{Name: name, Location: domain.Point{Lat: lat, Lon: lon}}
	}

	parkEnd := end("Vaso Godziashvili Park", 41.725095, 44.755362)
	javakhishvili := end("Mikheil Javakhishvili Street", 41.70497, 44.791844)
	number69 := end("ჯავახიშვილის #69", 41.711185, 44.800602)

	return domain.Itinerary{
		Start: at("2026-10-19 21:42:48"),
		End:   at("2026-10-19 22:43:21"),
		Legs: []domain.Leg{
			{
				Kind:   domain.KindWalk,
				From:   end("Origin", origin().Lat, origin().Lon),
				To:     parkEnd,
				Metres: 3.2,
			},
			{
				Kind:        domain.KindBus,
				From:        parkEnd,
				To:          javakhishvili,
				RouteNumber: "310",
				Start: at(
					"2026-10-19 21:42:51",
				),
				End:    at("2026-10-19 21:59:48"),
				Metres: 4913.08,
				Stops:  12,
			},
			{Kind: domain.KindBus, From: javakhishvili, To: number69, RouteNumber: "532", Stops: 4},
			{
				Kind: domain.KindWalk,
				From: number69,
				To:   end("Destination", destination().Lat, destination().Lon),
			},
		},
	}
}

// TTCTRANS-SC-033: Each end of a ride gets the stop at its coordinates, and each ride
// of a minibus route gets the kind minibus. A point outside Tbilisi is refused before
// the plan.
func TestPlanTrip(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{itineraries: []domain.Itinerary{fixtureItinerary()}}
	use := app.NewPlanTrip(newNetwork(), planner, fixedNow("2026-10-19 22:00:00"))

	got, err := use.Plan(t.Context(), app.TripQuery{From: origin(), To: destination()})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	legs := got.Itineraries[0].Legs

	type ends struct{ From, To domain.StopID }

	gotEnds := make([]ends, 0, len(legs))
	for _, leg := range legs {
		gotEnds = append(gotEnds, ends{From: leg.From.Stop, To: leg.To.Stop})
	}

	want := []ends{{"", park}, {park, "1:2372"}, {"1:2372", "1:20148"}, {"1:20148", ""}}
	if diff := cmp.Diff(want, gotEnds); diff != "" {
		t.Errorf("stops mismatch (-want +got):\n%s", diff)
	}

	if legs[1].Kind != domain.KindBus || legs[2].Kind != domain.KindMinibus {
		t.Errorf("kinds = %s, %s, want bus, minibus", legs[1].Kind, legs[2].Kind)
	}

	_, err = use.Plan(
		t.Context(),
		app.TripQuery{From: origin(), To: domain.Point{Lat: 40, Lon: 40}},
	)

	outside, ok := errors.AsType[*app.OutsideTbilisiError](err)
	if !ok || outside.Field != "to" {
		t.Errorf("Plan() error = %v, want *OutsideTbilisiError of to", err)
	}

	if calls := planner.calls.Load(); calls != 1 {
		t.Errorf("the planner got %d calls, want 1", calls)
	}
}

// TTCTRANS-SC-034: A time with no date takes the date in Tbilisi. depart_at and
// arrive_by need a time. The defaults are every kind and the fastest trip.
func TestPlanTripTiming(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{}
	use := app.NewPlanTrip(newNetwork(), planner, fixedNow("2026-10-19 22:00:00"))
	nine := mustClock("09:00")

	if _, err := use.Plan(t.Context(), app.TripQuery{
		From: origin(), To: destination(), Timing: domain.TimingArriveBy, Time: &nine,
		Language: kernel.LanguageEnglish,
	}); err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	want := app.TripRequest{
		From:       origin(),
		To:         destination(),
		Timing:     domain.TimingArriveBy,
		At:         at("2026-10-19 09:00:00"),
		Kinds:      domain.RouteKinds(),
		Preference: domain.PreferenceFastest,
		Language:   kernel.LanguageEnglish,
	}
	if diff := cmp.Diff(want, planner.request); diff != "" {
		t.Errorf("request mismatch (-want +got):\n%s", diff)
	}

	_, err := use.Plan(
		t.Context(),
		app.TripQuery{From: origin(), To: destination(), Timing: domain.TimingDepartAt},
	)
	if !errors.Is(err, domain.ErrInvalidClockTime) {
		t.Errorf("Plan() error = %v, want ErrInvalidClockTime", err)
	}

	if _, err := use.Plan(
		t.Context(),
		app.TripQuery{From: origin(), To: destination()},
	); err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if !planner.request.At.IsZero() || planner.request.Timing != domain.TimingLeaveNow {
		t.Errorf("request = %+v, want leave_now with no instant", planner.request)
	}
}

// TTCTRANS-SC-035: The places come up to the limit. An empty text is refused before
// the call.
func TestSearchPlaces(t *testing.T) {
	t.Parallel()

	places := &fakePlaces{places: make([]domain.Place, 8)}
	use := app.NewSearchPlaces(places)

	got, err := use.Search(t.Context(), app.PlaceQuery{Text: "Rustaveli", Limit: 5})
	if err != nil || len(got.Places) != 5 {
		t.Errorf("Search() = %d places, %v, want 5", len(got.Places), err)
	}

	if _, err := use.Search(
		t.Context(),
		app.PlaceQuery{Text: " ", Limit: 5},
	); !errors.Is(
		err,
		app.ErrEmptyQuery,
	) {
		t.Errorf("Search() error = %v, want ErrEmptyQuery", err)
	}

	if calls := places.calls.Load(); calls != 1 {
		t.Errorf("the finder got %d calls, want 1", calls)
	}
}
