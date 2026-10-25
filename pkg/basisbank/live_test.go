//go:build live

package basisbank_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/basisbank"
)

// BASRATE-SC-004: The real rate method of Basisbank has a rate of USD above zero.
func TestLiveRates(t *testing.T) {
	t.Parallel()

	days, err := basisbank.New().Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	for _, day := range days {
		buy, sell := day.XRates.KursBuy[dollar], day.XRates.KursSell[dollar]
		if buy == "" || sell == "" {
			continue
		}

		for name, price := range map[string]string{"Buy": buy, "Sell": sell} {
			if price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("Rates() = %+v, want a day with a rate of USD", days)
}
