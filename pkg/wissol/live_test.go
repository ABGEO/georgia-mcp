//go:build live

package wissol_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/wissol"
)

// WISFUEL-SC-003: The real Wissol answers one product or more, and each active product
// has a price.
func TestLiveFuelPrices(t *testing.T) {
	t.Parallel()

	prices, err := wissol.New().FuelPrices(t.Context())
	if err != nil {
		t.Fatalf("FuelPrices() error = %v", err)
	}

	if len(prices) == 0 {
		t.Fatal("FuelPrices() holds no product")
	}

	for _, price := range prices {
		if price.IsActive && (price.Price == "" || price.Price[0] == '0' || price.Price[0] == '-') {
			t.Errorf("%s: price = %q, want a price above zero", price.FuelTypeEng, price.Price)
		}
	}
}
