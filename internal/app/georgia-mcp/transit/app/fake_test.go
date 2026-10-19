package app_test

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	park       = "1:1000"
	tandzia    = "1:1001"
	route385   = "1:R29925"
	route472   = "1:minibusR25521"
	metroLine1 = "1:Metro_Metro_1"
)

func readAt() time.Time {
	return time.Date(2026, 10, 19, 22, 10, 24, 0, kernel.TbilisiZone())
}

func stop(id, code, english, georgian string, kind domain.Kind, lat, lon float64) domain.Stop {
	return domain.Stop{
		ID:       domain.StopID(id),
		Code:     code,
		Name:     domain.Name{Georgian: georgian, English: english},
		Kind:     kind,
		Location: domain.Point{Lat: lat, Lon: lon},
	}
}

func route(id, number, english, georgian string, kind domain.Kind) domain.Route {
	return domain.Route{
		ID:     domain.RouteID(id),
		Number: number,
		Name:   domain.Name{Georgian: georgian, English: english},
		Kind:   kind,
	}
}

// The stops and the routes of the TTC on 2026-10-19 that the scenarios name.
func fixtureStops() []domain.Stop {
	return append(stopsNearPark(), otherStops()...)
}

// stopsNearPark are the four stops within 500 m of 41.7251, 44.7554.
func stopsNearPark() []domain.Stop {
	return []domain.Stop{
		stop(
			park,
			"1000",
			"Vaso Godziashvili Park",
			"ვასო გოძიაშვილის ბაღი",
			domain.KindBus,
			41.725095,
			44.755362,
		),
		stop(
			tandzia,
			"1001",
			"Tandzia Street",
			"ტანძიის ქუჩა",
			domain.KindBus,
			41.725458,
			44.759042,
		),
		stop(
			"1:1054",
			"1054",
			"Republican Hospital",
			"რესპუბლიკური საავადმყოფო",
			domain.KindBus,
			41.726771,
			44.757888,
		),
		stop(
			"1:1055",
			"1055",
			"Medical University",
			"სამედიცინო უნივერსიტეტი",
			domain.KindBus,
			41.726303,
			44.753119,
		),
	}
}

func otherStops() []domain.Stop {
	return []domain.Stop{
		stop("1:gondola_1", "", "Rike", "რიყე", domain.KindCableCar, 41.692039, 44.810776),
		stop("1:metro_1_14", "", "M/S Isani", "მ/ს ისანი", domain.KindMetro, 41.686632, 44.84084),
		stop("1:1017", "1017", "M/S Isani", "მ/ს \"ისანი\"", domain.KindBus, 41.686728, 44.841467),
		stop("1:1151", "1151", "Vake Park", "ვაკის პარკი", domain.KindBus, 41.711802, 44.751126),
		stop("1:3898", "3898", "Vake Park", "ვაკის პარკი", domain.KindBus, 41.709869, 44.753604),
		stop(
			"1:2372",
			"2372",
			"Mikheil Javakhishvili Street",
			"მიხეილ ჯავახიშვილის ქუჩა",
			domain.KindBus,
			41.70497,
			44.791844,
		),
		stop(
			"1:20148",
			"20148",
			"ჯავახიშვილის #69",
			"ჯავახიშვილის #69",
			domain.KindBus,
			41.711185,
			44.800602,
		),
	}
}

func fixtureRoutes() []domain.Route {
	return []domain.Route{
		route(
			metroLine1,
			"1",
			"M/S Akhmeteli - M/S Varketili",
			"მ/ს ახმეტელი - მ/ს ვარკეთილი",
			domain.KindMetro,
		),
		route(
			"1:Metro_Metro_2",
			"2",
			"M/S Station Square - M/S State University",
			"მ/ს სადგურის მოედანი - მ/ს სახელმწიფო უნივერსიტეტი",
			domain.KindMetro,
		),
		route(
			"1:Gondola_Gondola_1",
			"1",
			"Rike - Narikala",
			"რიყე - ნარიყალა",
			domain.KindCableCar,
		),
		route(
			"1:R98476",
			"310",
			"Varketili M/D-4 - State University H/B",
			"ვარკეთილის IV მ/რ - სახ. უნივერსიტეტის მ/კ",
			domain.KindBus,
		),
		route(route385, "385", "Vake-Saburtalo", "ვაკე - საბურთალო", domain.KindBus),
		route(
			route472,
			"472",
			"",
			"სახელმწიფო უნივერსიტეტის მე-10 კორპუსი - ვლადიმერ ლობჟანიძის ქ.",
			domain.KindMinibus,
		),
		route(
			"1:minibusR24571",
			"532",
			"",
			"დუმბაძის გამზ. - თავისუფლების მოედ.",
			domain.KindMinibus,
		),
	}
}

// fakeNetwork answers the fixture lists and counts its calls.
type fakeNetwork struct {
	stops  []domain.Stop
	routes []domain.Route
	calls  atomic.Int32
}

func newNetwork() *fakeNetwork {
	return &fakeNetwork{stops: fixtureStops(), routes: fixtureRoutes()}
}

func (f *fakeNetwork) Stops(context.Context) (app.StopList, error) {
	f.calls.Add(1)

	return app.StopList{Stops: f.stops, ReadAt: readAt()}, nil
}

func (f *fakeNetwork) Routes(context.Context) (app.RouteList, error) {
	f.calls.Add(1)

	return app.RouteList{Routes: f.routes, ReadAt: readAt()}, nil
}

// fakeArrivals answers fixed arrivals and keeps the last request.
type fakeArrivals struct {
	arrivals []domain.Arrival
	request  app.ArrivalRequest
	calls    atomic.Int32
}

func (f *fakeArrivals) Arrivals(
	_ context.Context,
	request app.ArrivalRequest,
) (app.ArrivalList, error) {
	f.calls.Add(1)
	f.request = request

	return app.ArrivalList{Arrivals: f.arrivals, ReadAt: readAt()}, nil
}

// fakeRoutes answers fixed directions and a fixed timetable.
type fakeRoutes struct {
	directions map[domain.RouteID][]domain.Direction
	timetable  domain.Timetable
	asked      int
}

func (f *fakeRoutes) Directions(_ context.Context, id domain.RouteID) (app.DirectionList, error) {
	return app.DirectionList{Directions: f.directions[id], ReadAt: readAt()}, nil
}

func (f *fakeRoutes) Timetable(
	_ context.Context,
	_ domain.RouteID,
	direction int,
) (app.TimetableRead, error) {
	f.asked = direction

	return app.TimetableRead{Timetable: f.timetable, ReadAt: readAt()}, nil
}

// fakePlanner answers fixed itineraries and keeps the last request.
type fakePlanner struct {
	itineraries []domain.Itinerary
	request     app.TripRequest
	calls       atomic.Int32
}

func (f *fakePlanner) Plan(_ context.Context, request app.TripRequest) (app.TripPlan, error) {
	f.calls.Add(1)
	f.request = request

	return app.TripPlan{Itineraries: f.itineraries, ReadAt: readAt()}, nil
}

// fakePlaces answers fixed places and counts its calls.
type fakePlaces struct {
	places []domain.Place
	calls  atomic.Int32
}

func (f *fakePlaces) Places(context.Context, string, kernel.Language) (app.PlaceList, error) {
	f.calls.Add(1)

	return app.PlaceList{Places: f.places, ReadAt: readAt()}, nil
}

func fixedNow(text string) func() time.Time {
	return func() time.Time {
		now, err := time.ParseInLocation(time.DateTime, text, kernel.TbilisiZone())
		if err != nil {
			panic(err)
		}

		return now
	}
}

func mustDate(text string) kernel.Date {
	date, err := kernel.ParseDate(text)
	if err != nil {
		panic(err)
	}

	return date
}

func mustClock(text string) domain.ClockTime {
	clock, err := domain.ParseClockTime(text)
	if err != nil {
		panic(err)
	}

	return clock
}

func stopIDs(stops []domain.Stop) []domain.StopID {
	ids := make([]domain.StopID, 0, len(stops))
	for _, s := range stops {
		ids = append(ids, s.ID)
	}

	return ids
}
