//go:build live

package terabank_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/terabank"
)

// TERARATE-SC-004: The real API of Terabank quotes USD above zero.
func TestLiveCrossRates(t *testing.T) {
	t.Parallel()

	rates, err := terabank.New().CrossRates(t.Context())
	if err != nil {
		t.Fatalf("CrossRates() error = %v", err)
	}

	for _, rate := range rates {
		if rate.ISO != dollar {
			continue
		}

		for name, price := range map[string]string{"Buy": rate.Buy.String(), "Sell": rate.Sell.String()} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("CrossRates() = %+v, want an item of USD", rates)
}
