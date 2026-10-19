package app_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func routeIDs(routes []domain.Route) []domain.RouteID {
	ids := make([]domain.RouteID, 0, len(routes))
	for _, r := range routes {
		ids = append(ids, r.ID)
	}

	return ids
}

func metroDirections() map[domain.RouteID][]domain.Direction {
	forward := make([]domain.StopID, 16)
	backward := make([]domain.StopID, 16)

	for i := range 16 {
		forward[i] = domain.StopID("1:metro_1_" + string(rune('a'+i)))
		backward[15-i] = forward[i]
	}

	return map[domain.RouteID][]domain.Direction{
		metroLine1: {
			{ID: 1, Headsign: domain.Name{English: "Varketili"}, Stops: forward},
			{ID: 0, Headsign: domain.Name{English: "Akhmeteli Theatre"}, Stops: backward},
		},
		route385: {
			{
				ID:       1,
				Headsign: domain.Name{English: "Vake - Saburtalo"},
				Stops:    []domain.StopID{tandzia, park, tandzia},
			},
		},
	}
}

// TTCTRANS-SC-028: A search matches a route number exactly, and a name in either
// language. With no text, it lists every route by kind, then by number.
func TestSearchRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query app.RouteSearch
		want  []domain.RouteID
	}{
		{
			name:  "a number",
			query: app.RouteSearch{Text: "385", Limit: 20},
			want:  []domain.RouteID{route385},
		},
		{
			name:  "a number of two routes",
			query: app.RouteSearch{Text: "1", Limit: 2},
			want:  []domain.RouteID{metroLine1, "1:Gondola_Gondola_1"},
		},
		{
			name:  "no text and a kind",
			query: app.RouteSearch{Kind: domain.KindMetro, Limit: 20},
			want:  []domain.RouteID{metroLine1, "1:Metro_Metro_2"},
		},
		{
			name:  "no text",
			query: app.RouteSearch{Limit: 20},
			want: []domain.RouteID{
				"1:R98476", route385, route472, "1:minibusR24571",
				metroLine1, "1:Metro_Metro_2", "1:Gondola_Gondola_1",
			},
		},
		{
			name:  "a Georgian name",
			query: app.RouteSearch{Text: "ვაკე", Language: kernel.LanguageEnglish, Limit: 20},
			want:  []domain.RouteID{route385},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := app.NewSearchRoutes(newNetwork()).Search(t.Context(), tt.query)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, routeIDs(got.Routes)); diff != "" {
				t.Errorf("Search() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TTCTRANS-SC-029: A number of two routes needs a kind. An unknown route is refused.
func TestGetRouteResolvesTheRoute(t *testing.T) {
	t.Parallel()

	use := app.NewGetRoute(newNetwork(), &fakeRoutes{directions: metroDirections()})

	_, err := use.Get(t.Context(), app.RouteQuery{Route: "1"})

	ambiguous, ok := errors.AsType[*app.AmbiguousRouteError](err)
	if !ok || !errors.Is(err, app.ErrAmbiguousRoute) || len(ambiguous.Routes) != 2 {
		t.Errorf("Get(1) error = %v, want *AmbiguousRouteError with 2 routes", err)
	}

	_, err = use.Get(t.Context(), app.RouteQuery{Route: "999"})
	if !errors.Is(err, app.ErrUnknownRoute) {
		t.Errorf("Get(999) error = %v, want ErrUnknownRoute", err)
	}
}

// TTCTRANS-SC-029: The metro line 1 has two directions of 16 stations each.
func TestGetRouteGivesEachDirection(t *testing.T) {
	t.Parallel()

	use := app.NewGetRoute(newNetwork(), &fakeRoutes{directions: metroDirections()})

	got, err := use.Get(t.Context(), app.RouteQuery{Route: "1", Kind: domain.KindMetro})
	if err != nil {
		t.Fatalf("Get(1, metro) error = %v", err)
	}

	if got.Route.ID != metroLine1 || len(got.Directions) != 2 ||
		len(got.Directions[0].Stops) != 16 || len(got.Directions[1].Stops) != 16 {
		t.Errorf("Get(1, metro) = %+v, want 2 directions of 16 stops", got)
	}
}

// TTCTRANS-SC-029: The stops of a direction come in order, with the facts of the list.
func TestGetRouteGivesTheStopsInOrder(t *testing.T) {
	t.Parallel()

	use := app.NewGetRoute(newNetwork(), &fakeRoutes{directions: metroDirections()})

	loop, err := use.Get(t.Context(), app.RouteQuery{Route: route385})
	if err != nil {
		t.Fatalf("Get(1:R29925) error = %v", err)
	}

	stops := loop.Directions[0].Stops
	if diff := cmp.Diff([]domain.StopID{tandzia, park, tandzia}, stopIDs(stops)); diff != "" {
		t.Errorf("stops mismatch (-want +got):\n%s", diff)
	}

	if stops[1].Name.English != "Vaso Godziashvili Park" {
		t.Errorf("stop = %+v, want the stop of the list", stops[1])
	}
}
