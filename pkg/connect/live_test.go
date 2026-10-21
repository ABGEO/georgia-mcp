//go:build live

package connect_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/connect"
)

// CONFUEL-SC-003: The real Connect gives one product or more, and each product has a
// label and a standard price.
func TestLiveCurrent(t *testing.T) {
	t.Parallel()

	snapshot, err := connect.New().Current(t.Context())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if len(snapshot.Prices) == 0 {
		t.Fatal("Prices is empty, want one product or more")
	}

	for key, price := range snapshot.Prices {
		if price.Label == "" || price.Standard == "" {
			t.Errorf("%s = %+v, want a label and a standard price", key, price)
		}
	}
}
