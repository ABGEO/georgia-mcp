//go:build live

package inteli_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/inteli"
)

// INTLRATE-SC-004: The real home page of Inteli Express has a rate of USD in GEL above zero.
func TestLiveRates(t *testing.T) {
	t.Parallel()

	rows, err := inteli.New().Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	for _, row := range rows {
		if row.Base != lari || row.Currency != dollar {
			continue
		}

		for name, price := range map[string]string{"Buy": row.Buy, "Sell": row.Sell} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("Rates() = %+v, want a row of USD in GEL", rows)
}
