//go:build live

package crystal_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/crystal"
)

// CRYSRATE-SC-004: The real method of Crystal has a rate of USD in GEL above zero.
func TestLiveCashRates(t *testing.T) {
	t.Parallel()

	rates, err := crystal.New().CashRates(t.Context())
	if err != nil {
		t.Fatalf("CashRates() error = %v", err)
	}

	for _, rate := range rates {
		if rate.SourceISO != dollar || rate.TargetISO != lari {
			continue
		}

		for name, price := range map[string]string{
			"BuyRate":  rate.BuyRate.String(),
			"SellRate": rate.SellRate.String(),
		} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("CashRates() = %+v, want an item of USD in GEL", rates)
}
