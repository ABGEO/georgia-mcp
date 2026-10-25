//go:build live

package scapp_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/scapp"
)

// SCAPRATE-SC-004: The real rate page of Scapp has a rate of USD above zero.
func TestLiveRates(t *testing.T) {
	t.Parallel()

	rows, err := scapp.New().Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	for _, row := range rows {
		if row.Unit != dollarUnit {
			continue
		}

		for name, price := range map[string]string{"Buy": row.Buy, "Sell": row.Sell} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("Rates() = %+v, want a row of 1 USD", rows)
}
