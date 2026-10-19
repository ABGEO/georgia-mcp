//go:build live

package rompetrol_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/rompetrol"
)

// ROMFUEL-SC-004: The real Rompetrol shows one row or more, and each row has a product
// and a price.
func TestLivePrices(t *testing.T) {
	t.Parallel()

	rows, err := rompetrol.New().Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	if len(rows) == 0 {
		t.Fatal("Prices() holds no row")
	}

	for _, row := range rows {
		if row.Product == "" || row.Price == "" {
			t.Errorf("row = %q, want a product and a price", row)
		}
	}
}
