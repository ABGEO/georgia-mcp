//go:build live

package halyk_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/halyk"
)

// HALRATE-SC-004: The real rate page of Halyk Bank has a rate of USD above zero.
func TestLiveRates(t *testing.T) {
	t.Parallel()

	rows, err := halyk.New().Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	for _, row := range rows {
		if row.Currency != dollar {
			continue
		}

		for name, price := range map[string]string{"Buy": row.Buy, "Sell": row.Sell} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("Rates() = %+v, want a row of USD", rows)
}
