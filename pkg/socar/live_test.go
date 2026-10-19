//go:build live

package socar_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/socar"
)

// SOCFUEL-SC-003: The real SOCAR reports a success with one product or more, and each
// product has a code and a price.
func TestLiveCurrentPrices(t *testing.T) {
	t.Parallel()

	prices, err := socar.New().CurrentPrices(t.Context())
	if err != nil {
		t.Fatalf("CurrentPrices() error = %v", err)
	}

	if prices.Status != "1" || len(prices.Results) == 0 {
		t.Fatalf(
			"Status = %q with %d products, want 1 with one or more",
			prices.Status,
			len(prices.Results),
		)
	}

	for _, price := range prices.Results {
		if price.FuelCode == "" || price.FuelUnitPrice == "" {
			t.Errorf("product = %+v, want a code and a price", price)
		}
	}
}
