package app_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// TTCTRANS-SC-023: A search matches a stop code exactly, and a name in either
// language. A kind limits the search.
func TestSearchStops(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query app.StopSearch
		want  []domain.StopID
	}{
		{
			name:  "a code",
			query: app.StopSearch{Text: "1000", Limit: 10},
			want:  []domain.StopID{park},
		},
		{
			name:  "a part of an English name",
			query: app.StopSearch{Text: "  GODZIASHVILI ", Limit: 10},
			want:  []domain.StopID{park},
		},
		{
			name:  "a Georgian name with English answers",
			query: app.StopSearch{Text: "ვაკის პარკი", Language: kernel.LanguageEnglish, Limit: 10},
			want:  []domain.StopID{"1:1151", "1:3898"},
		},
		{
			name:  "a kind",
			query: app.StopSearch{Text: "isani", Kind: domain.KindMetro, Limit: 10},
			want:  []domain.StopID{"1:metro_1_14"},
		},
		{
			name:  "a limit",
			query: app.StopSearch{Text: "isani", Limit: 1},
			want:  []domain.StopID{"1:1017"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := app.NewSearchStops(newNetwork()).Search(t.Context(), tt.query)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, stopIDs(got.Stops)); diff != "" {
				t.Errorf("Search() mismatch (-want +got):\n%s", diff)
			}

			if !got.ReadAt.Equal(readAt()) {
				t.Errorf("ReadAt = %v, want %v", got.ReadAt, readAt())
			}
		})
	}
}

// TTCTRANS-SC-023: A code that equals the text comes before a name that holds it.
func TestSearchStopsPutsTheCodeFirst(t *testing.T) {
	t.Parallel()

	network := newNetwork()
	network.stops = append(network.stops, stop(
		"1:9", "9", "Street 1000", "ქუჩა 1000", domain.KindBus, 41.72, 44.76,
	))

	got, err := app.NewSearchStops(network).
		Search(t.Context(), app.StopSearch{Text: "1000", Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if diff := cmp.Diff([]domain.StopID{park, "1:9"}, stopIDs(got.Stops)); diff != "" {
		t.Errorf("Search() mismatch (-want +got):\n%s", diff)
	}
}

// TTCTRANS-SC-023: An empty text is refused before a read of the stops.
func TestSearchStopsRefusesAnEmptyText(t *testing.T) {
	t.Parallel()

	network := newNetwork()

	_, err := app.NewSearchStops(network).
		Search(t.Context(), app.StopSearch{Text: "   ", Limit: 10})
	if !errors.Is(err, app.ErrEmptyQuery) {
		t.Errorf("Search() error = %v, want ErrEmptyQuery", err)
	}

	if calls := network.calls.Load(); calls != 0 {
		t.Errorf("the network got %d calls, want 0", calls)
	}
}

// TTCTRANS-SC-024: The stops within a distance come nearest first, with the distance.
// A point outside Tbilisi is refused.
func TestListNearbyStops(t *testing.T) {
	t.Parallel()

	nearby := app.NewListNearbyStops(newNetwork())
	point := domain.Point{Lat: 41.7251, Lon: 44.7554}

	got, err := nearby.List(t.Context(), app.NearbyQuery{Point: point, Radius: 300, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	type found struct {
		ID     domain.StopID
		Metres int
	}

	gotFound := make([]found, 0, len(got.Stops))
	for _, s := range got.Stops {
		gotFound = append(gotFound, found{ID: s.Stop.ID, Metres: s.Metres})
	}

	want := []found{{ID: park, Metres: 3}, {ID: "1:1055", Metres: 232}, {ID: "1:1054", Metres: 278}}
	if diff := cmp.Diff(want, gotFound); diff != "" {
		t.Errorf("List() mismatch (-want +got):\n%s", diff)
	}

	wide, err := nearby.List(t.Context(), app.NearbyQuery{Point: point, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(wide.Stops) != 4 || wide.Stops[3].Stop.ID != tandzia {
		t.Errorf("List() with no radius gives %+v, want 4 stops within 500 m", wide.Stops)
	}

	_, err = nearby.List(
		t.Context(),
		app.NearbyQuery{Point: domain.Point{Lat: 40, Lon: 40}, Limit: 10},
	)
	if !errors.Is(err, app.ErrOutsideTbilisi) {
		t.Errorf("List() error = %v, want ErrOutsideTbilisi", err)
	}
}
