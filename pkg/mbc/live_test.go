//go:build live

package mbc_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/mbc"
)

// MBCRATE-SC-004: The real method of MBC has a rate of USD in GEL above zero.
func TestLiveCommercialRates(t *testing.T) {
	t.Parallel()

	rates, err := mbc.New().CommercialRates(t.Context())
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	for _, rate := range rates {
		if rate.FromCcy != dollar || rate.ToCcy != lari {
			continue
		}

		for name, price := range map[string]string{"Buy": rate.Buy.String(), "Sell": rate.Sell.String()} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("CommercialRates() = %+v, want an item of USD in GEL", rates)
}
