//go:build live

package kursi_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/kursi"
)

// KURSRATE-SC-004: The real API of Kursi.ge quotes USD against the lari above zero.
func TestLiveCurrencies(t *testing.T) {
	t.Parallel()

	pairs, err := kursi.New().Currencies(t.Context())
	if err != nil {
		t.Fatalf("Currencies() error = %v", err)
	}

	for _, pair := range pairs {
		if pair.BaseCurrencyCode != "GEL" || pair.SecondaryCurrencyCode != dollar {
			continue
		}

		for name, price := range map[string]string{"BuyRate": pair.BuyRate.String(), "SellRate": pair.SellRate.String()} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("Currencies() = %+v, want the pair of GEL and USD", pairs)
}
