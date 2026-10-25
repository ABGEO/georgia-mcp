//go:build live

package credo_test

import (
	"os"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/credo"
)

// CREDRATE-SC-004: The real method of Credo Bank has a rate of USD against GEL above
// zero.
func TestLiveCurrentRates(t *testing.T) {
	t.Parallel()

	key := os.Getenv("GEOMCP_RATES_CREDO_API_KEY")
	if key == "" {
		t.Skip("GEOMCP_RATES_CREDO_API_KEY is empty")
	}

	rates, err := credo.New(credo.WithAPIKey(key)).
		CurrentRates(t.Context(), credo.MethodCommercial, time.Now())
	if err != nil {
		t.Fatalf("CurrentRates() error = %v", err)
	}

	for _, rate := range rates {
		if rate.FromCcy != dollar || rate.ToCcy != "GEL" {
			continue
		}

		for name, price := range map[string]string{"Buy": rate.Buy.String(), "Sell": rate.Sell.String()} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("CurrentRates() = %+v, want an item of USD against GEL", rates)
}
