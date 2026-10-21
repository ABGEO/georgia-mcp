//go:build live

package lukoil_test

import (
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/lukoil"
)

// LUKFUEL-SC-003: The real Lukoil gives a head of 3 cells or more, and the last cell of
// each row is an instant.
func TestLivePriceHistory(t *testing.T) {
	t.Parallel()

	table, err := lukoil.New().PriceHistory(t.Context())
	if err != nil {
		t.Fatalf("PriceHistory() error = %v", err)
	}

	if len(table.Header) < 3 {
		t.Fatalf("Header = %q, want 3 cells or more", table.Header)
	}

	for _, row := range table.Rows {
		if _, err := time.Parse(time.DateTime, row[len(row)-1]); err != nil {
			t.Errorf("row %q: the last cell is not an instant: %v", row, err)
		}
	}
}
