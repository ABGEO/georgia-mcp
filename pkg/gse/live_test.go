//go:build live

package gse_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/gse"
)

func hasBlock(export gse.Export, name string) bool {
	return slices.ContainsFunc(export.Blocks, func(b gse.Block) bool {
		return strings.TrimSpace(b.Name) == name && len(b.Entries) == 1
	})
}

// GSEGRID-SC-025: The grid of 12:00 yesterday holds the consumption, and the frequency of
// the same minute holds the frequency.
func TestLiveGrid(t *testing.T) {
	t.Parallel()

	zone, err := time.LoadLocation("Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}

	day := time.Now().In(zone).AddDate(0, 0, -1)
	at := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, zone)
	client := gse.New()

	grid, err := client.Grid(t.Context(), at)
	if err != nil {
		t.Fatalf("Grid() error = %v", err)
	}

	if !hasBlock(grid, "TOTAL CONSUMPTION") {
		t.Errorf("Blocks = %+v, want TOTAL CONSUMPTION with one entry", grid.Blocks)
	}

	frequency, err := client.Frequency(t.Context(), at)
	if err != nil {
		t.Fatalf("Frequency() error = %v", err)
	}

	if !hasBlock(frequency, "FREQUENCY") {
		t.Errorf("Blocks = %+v, want FREQUENCY with one entry", frequency.Blocks)
	}
}
