//go:build live

package tbc_test

import (
	"os"
	"slices"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/tbc"
)

// TBCRATE-SC-004: The real TBC answers with the rates of USD and JPY against GEL.
func TestLiveCommercialRates(t *testing.T) {
	t.Parallel()

	key := os.Getenv("GEOMCP_RATES_TBC_API_KEY")
	if key == "" {
		t.Skip("GEOMCP_RATES_TBC_API_KEY is empty")
	}

	rates, err := tbc.New(tbc.WithAPIKey(key)).CommercialRates(t.Context(), []string{"USD", "JPY"})
	if err != nil {
		t.Fatalf("CommercialRates() error = %v", err)
	}

	if rates.Base != "GEL" {
		t.Errorf("Base = %q, want GEL", rates.Base)
	}

	codes := make([]string, 0, len(rates.CommercialRatesList))
	for _, rate := range rates.CommercialRatesList {
		codes = append(codes, rate.Currency)
	}

	slices.Sort(codes)

	if !slices.Equal(codes, []string{"JPY", "USD"}) {
		t.Errorf("currencies = %v, want [JPY USD]", codes)
	}
}
