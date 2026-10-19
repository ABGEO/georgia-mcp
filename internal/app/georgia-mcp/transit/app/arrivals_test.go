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

func arrival(number string, minutes int) domain.Arrival {
	return domain.Arrival{
		RouteNumber: number,
		Kind:        domain.KindBus,
		Minutes:     minutes,
		Live:        true,
		Expected:    readAt().Add(time.Duration(minutes) * time.Minute),
	}
}

// TTCTRANS-SC-025: A stop resolves by identifier or by code. An unknown stop is
// refused before a read of the arrivals.
func TestListArrivalsResolvesTheStop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text    string
		want    domain.StopID
		wantErr error
	}{
		{text: "1000", want: park},
		{text: park, want: park},
		{text: "1:gondola_1", want: "1:gondola_1"},
		{text: "999999", wantErr: app.ErrUnknownStop},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			t.Parallel()

			arrivals := &fakeArrivals{}
			use := app.NewListArrivals(newNetwork(), arrivals)

			got, err := use.List(t.Context(), app.ArrivalQuery{Stop: tt.text, Limit: 10})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("List(%q) error = %v, want %v", tt.text, err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if calls := arrivals.calls.Load(); calls != 0 {
					t.Errorf("the arrival source got %d calls, want 0", calls)
				}

				return
			}

			if got.Stop.ID != tt.want || arrivals.request.Stop != tt.want {
				t.Errorf(
					"stop = %s, asked %s, want %s",
					got.Stop.ID,
					arrivals.request.Stop,
					tt.want,
				)
			}
		})
	}
}

// TTCTRANS-SC-026: A route asks the arrivals of that route, as many as the limit, and
// each arrival of a minibus route has the kind minibus.
func TestListArrivalsOfARoute(t *testing.T) {
	t.Parallel()

	arrivals := &fakeArrivals{arrivals: []domain.Arrival{
		arrival("472", 5),
		arrival("472", 19),
	}}

	got, err := app.NewListArrivals(newNetwork(), arrivals).List(t.Context(), app.ArrivalQuery{
		Stop: "1000", Route: "472", Language: kernel.LanguageEnglish, Limit: 4,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	wantRequest := app.ArrivalRequest{
		Stop: park, Route: route472, Max: 4, Language: kernel.LanguageEnglish,
	}
	if diff := cmp.Diff(wantRequest, arrivals.request); diff != "" {
		t.Errorf("request mismatch (-want +got):\n%s", diff)
	}

	for _, a := range got.Arrivals {
		if a.Kind != domain.KindMinibus {
			t.Errorf("arrival %+v has the kind %s, want minibus", a, a.Kind)
		}
	}

	if got.Route == nil || got.Route.ID != route472 {
		t.Errorf("Route = %+v, want 472", got.Route)
	}

	arrivals.arrivals = nil

	empty, err := app.NewListArrivals(newNetwork(), arrivals).List(
		t.Context(), app.ArrivalQuery{Stop: "1000", Route: "472", Limit: 4},
	)
	if err != nil || len(empty.Arrivals) != 0 {
		t.Errorf("List() = %+v, %v, want no arrival and no error", empty.Arrivals, err)
	}
}

// TTCTRANS-SC-026: An unknown route is refused.
func TestListArrivalsRefusesAnUnknownRoute(t *testing.T) {
	t.Parallel()

	_, err := app.NewListArrivals(newNetwork(), &fakeArrivals{}).List(
		t.Context(), app.ArrivalQuery{Stop: "1000", Route: "999", Limit: 4},
	)
	if !errors.Is(err, app.ErrUnknownRoute) {
		t.Errorf("List() error = %v, want ErrUnknownRoute", err)
	}
}

// TTCTRANS-SC-027: With no route, the answer holds the soonest arrivals up to the
// limit, and a tie sorts by route number.
func TestListArrivalsSortsAndCuts(t *testing.T) {
	t.Parallel()

	arrivals := &fakeArrivals{arrivals: []domain.Arrival{
		arrival("531", 1),
		arrival("551", 3),
		arrival("472", 5),
		arrival("385", 8),
		arrival("310", 8),
		arrival("300", 6),
		arrival("305", 7),
		arrival("349", 13),
	}}

	got, err := app.NewListArrivals(newNetwork(), arrivals).List(
		t.Context(), app.ArrivalQuery{Stop: "1000", Limit: 5},
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if arrivals.request.Route != "" || arrivals.request.Max != 0 {
		t.Errorf("request = %+v, want no route and no maximum", arrivals.request)
	}

	numbers := make([]string, 0, len(got.Arrivals))
	for _, a := range got.Arrivals {
		numbers = append(numbers, a.RouteNumber)
	}

	if diff := cmp.Diff([]string{"531", "551", "472", "300", "305"}, numbers); diff != "" {
		t.Errorf("arrivals mismatch (-want +got):\n%s", diff)
	}

	if got.Arrivals[2].Kind != domain.KindMinibus || got.Arrivals[3].Kind != domain.KindBus {
		t.Errorf("kinds = %s, %s, want minibus, bus", got.Arrivals[2].Kind, got.Arrivals[3].Kind)
	}

	arrivals.arrivals = []domain.Arrival{
		arrival("385", 8),
		arrival("310", 8),
	}

	tie, err := app.NewListArrivals(newNetwork(), arrivals).List(
		t.Context(), app.ArrivalQuery{Stop: "1000", Limit: 5},
	)
	if err != nil || tie.Arrivals[0].RouteNumber != "310" {
		t.Errorf("List() = %+v, %v, want 310 before 385", tie.Arrivals, err)
	}
}
