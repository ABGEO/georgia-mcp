package napr_test

import (
	"math"
	"testing"

	napradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/napr"
)

// NAPRPARC-SC-014: The centre of a polygon is its area centroid. A text that is not a
// polygon fails.
func TestCentre(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wkt      string
		lon, lat float64
	}{
		{name: "square", wkt: "POLYGON ((0 0, 1 0, 1 1, 0 1, 0 0))", lon: 0.5, lat: 0.5},
		{
			name: "two squares",
			wkt:  "MULTIPOLYGON (((0 0, 1 0, 1 1, 0 1, 0 0)), ((2 0, 3 0, 3 1, 2 1, 2 0)))",
			lon:  1.5, lat: 0.5,
		},
		{
			name: "a hole is left out",
			wkt:  "POLYGON ((0 0, 2 0, 2 2, 0 2, 0 0), (0 0, 1 0, 1 1, 0 1, 0 0))",
			lon:  1, lat: 1,
		},
		{name: "a line", wkt: "POLYGON ((0 0, 1 1, 2 2, 0 0))", lon: 1, lat: 1},
	}

	for _, tt := range tests {
		lon, lat, err := napradapter.Centre(tt.wkt)
		if err != nil || math.Abs(lon-tt.lon) > 1e-9 || math.Abs(lat-tt.lat) > 1e-9 {
			t.Errorf(
				"%s: centre() = %v, %v, %v, want %v, %v",
				tt.name,
				lon,
				lat,
				err,
				tt.lon,
				tt.lat,
			)
		}
	}

	for _, bad := range []string{"POINT (1 2)", "", "POLYGON ((a b))", "POLYGON (())", "POLYGON ((1 2, 3"} {
		if _, _, err := napradapter.Centre(bad); err == nil {
			t.Errorf("centre(%q) error = nil, want an error", bad)
		}
	}
}

// NAPRPARC-SC-014: The reader never panics, and a centre is a number.
func FuzzCentre(f *testing.F) {
	for _, seed := range []string{
		"POLYGON ((0 0, 1 0, 1 1, 0 1, 0 0))",
		"MULTIPOLYGON (((0 0, 1 0, 1 1, 0 0)))",
		"POLYGON ((",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, wkt string) {
		lon, lat, err := napradapter.Centre(wkt)
		if err == nil &&
			(math.IsNaN(lon) || math.IsNaN(lat) || math.IsInf(lon, 0) || math.IsInf(lat, 0)) {
			t.Errorf("centre(%q) = %v, %v, want numbers", wkt, lon, lat)
		}
	})
}
