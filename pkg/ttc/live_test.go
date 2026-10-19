//go:build live

package ttc_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/ttc"
)

// TTCTRANS-SC-008: The real TTC lists more than 2,000 stops, and 1:1000 is Vaso
// Godziashvili Park.
func TestLiveStops(t *testing.T) {
	t.Parallel()

	key := os.Getenv("GEOMCP_TRANSIT_TTC_API_KEY")
	if key == "" {
		t.Skip("GEOMCP_TRANSIT_TTC_API_KEY is empty")
	}

	stops, err := ttc.New(ttc.WithAPIKey(key)).Stops(t.Context(), "en")
	if err != nil {
		t.Fatalf("Stops() error = %v", err)
	}

	if len(stops) <= 2000 {
		t.Fatalf("Stops() gives %d stops, want more than 2000", len(stops))
	}

	for _, stop := range stops {
		if stop.ID == "1:1000" && stop.Name != "Vaso Godziashvili Park" {
			t.Errorf("1:1000 = %q, want Vaso Godziashvili Park", stop.Name)
		}
	}
}
