//go:build live

package libertyrates_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/libertyrates"
)

// LIBRATE-SC-004: The real home page of Liberty has a commercial rate of USD above zero.
func TestLiveRates(t *testing.T) {
	t.Parallel()

	rows, err := libertyrates.New().Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	for _, row := range rows {
		if row.Currency != dollar {
			continue
		}

		for name, price := range map[string]string{"Buy": row.Commercial.Buy, "Sell": row.Commercial.Sell} {
			if price == "" || price[0] == '0' || price[0] == '-' {
				t.Errorf("Commercial.%s = %q, want a price above zero", name, price)
			}
		}

		return
	}

	t.Errorf("Rates() = %+v, want a row of USD", rows)
}
