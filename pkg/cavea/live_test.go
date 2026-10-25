//go:build live

package cavea_test

import (
	"slices"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/cavea"
)

// CAVCINE-SC-025: The real Cavea lists East Point, publishes a session, and gives the
// seats of the first session of the last date of the page.
func TestLiveCavea(t *testing.T) {
	t.Parallel()

	client := cavea.New()

	theaters, err := client.Theaters(t.Context())
	if err != nil {
		t.Fatalf("Theaters() error = %v", err)
	}

	if !slices.ContainsFunc(theaters, func(theater cavea.Theater) bool {
		return theater.NameEng == "Cavea East Point"
	}) {
		t.Errorf("Theaters() = %+v, want Cavea East Point", theaters)
	}

	groups, err := client.Sessions(t.Context())
	if err != nil {
		t.Fatalf("Sessions() error = %v", err)
	}

	if len(groups) == 0 || len(groups[len(groups)-1].Movies) == 0 ||
		len(groups[len(groups)-1].Movies[0].Sessions) == 0 {
		t.Fatalf("Sessions() = %+v, want a session on the last date", groups)
	}

	session := groups[len(groups)-1].Movies[0].Sessions[0]

	seats, err := client.SeatMap(t.Context(), session.ID)
	if err != nil {
		t.Fatalf("SeatMap(%s) error = %v", session.ID, err)
	}

	if len(seats) == 0 {
		t.Errorf("SeatMap(%s) is empty, want one seat or more", session.ID)
	}
}
