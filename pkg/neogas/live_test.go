//go:build live

package neogas_test

import (
	"strconv"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/neogas"
)

// NEOFUEL-SC-003: The real Neogas gives one gas price or more, and the first is a number.
func TestLiveGasPrices(t *testing.T) {
	t.Parallel()

	prices, err := neogas.New().GasPrices(t.Context())
	if err != nil {
		t.Fatalf("GasPrices() error = %v", err)
	}

	if len(prices) == 0 {
		t.Fatal("GasPrices() is empty, want one value or more")
	}

	if _, err := strconv.ParseFloat(prices[0], 64); err != nil {
		t.Errorf("the first gas price %q is not a number: %v", prices[0], err)
	}
}
