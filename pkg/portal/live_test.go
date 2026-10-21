//go:build live

package portal_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/portal"
)

// PORFUEL-SC-003: The real Portal gives one entry or more, and each has a slug and a price.
func TestLiveFuelPrices(t *testing.T) {
	t.Parallel()

	prices, err := portal.New().FuelPrices(t.Context())
	if err != nil {
		t.Fatalf("FuelPrices() error = %v", err)
	}

	if len(prices) == 0 {
		t.Fatal("FuelPrices() is empty, want one entry or more")
	}

	for _, price := range prices {
		if price.FuelType.Slug == "" || price.LatestPrice == "" {
			t.Errorf("entry = %+v, want a slug and a price", price)
		}
	}
}
