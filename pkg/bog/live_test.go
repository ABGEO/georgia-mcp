//go:build live

package bog_test

import (
	"errors"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/bog"
)

// BOGRATE-SC-004: The real Bank of Georgia quotes USD above zero, and has no rate of GEL.
func TestLiveCommercialRate(t *testing.T) {
	t.Parallel()

	client := bog.New()

	rate, err := client.CommercialRate(t.Context(), dollar)
	if err != nil {
		t.Fatalf("CommercialRate(USD) error = %v", err)
	}

	for name, price := range map[string]string{"Buy": rate.Buy.String(), "Sell": rate.Sell.String()} {
		if price == "" || price[0] == '0' || price[0] == '-' {
			t.Errorf("%s = %q, want a price above zero", name, price)
		}
	}

	if _, err := client.CommercialRate(t.Context(), "GEL"); !errors.Is(err, bog.ErrNoRate) {
		t.Errorf("CommercialRate(GEL) error = %v, want ErrNoRate", err)
	}
}
