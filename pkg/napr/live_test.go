//go:build live

package napr_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/napr"
)

// NAPRPARC-SC-027: The parcel of Freedom Square has its detail and its shape, and a point
// on it has at least 5 parcels near.
func TestLiveParcel(t *testing.T) {
	t.Parallel()

	client := napr.New()

	results, err := client.Search(t.Context(), "01.15.05.066.020")
	if err != nil || len(results) == 0 {
		t.Fatalf("Search() = %v, %v, want a result", results, err)
	}

	label, ok := napr.Label(results[0].Details.InfoLink)
	if !ok {
		t.Fatalf("Label(%q) = false, want a label", results[0].Details.InfoLink)
	}

	infos, err := client.Parcel(t.Context(), label)
	if err != nil || len(infos) != 1 || infos[0].Detail.AreaMeasured != "618" {
		t.Errorf("Parcel() = %+v, %v, want the area 618", infos, err)
	}

	shapes, err := client.ParcelShape(t.Context(), label)
	if err != nil || len(shapes) != 1 || shapes[0].Shape == "" {
		t.Errorf("ParcelShape() = %+v, %v, want a shape", shapes, err)
	}

	parcels, err := client.ParcelsAt(t.Context(), 41.6934, 44.8015)
	if err != nil || len(parcels) < 5 {
		t.Errorf("ParcelsAt() = %d parcels, %v, want at least 5", len(parcels), err)
	}
}
