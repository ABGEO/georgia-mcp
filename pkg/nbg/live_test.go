//go:build live

package nbg_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/nbg"
)

// NBGRATE-SC-023: The real NBG answers with one set of 40 currencies or more.
func TestLiveRateSets(t *testing.T) {
	t.Parallel()

	sets, err := nbg.New().RateSets(t.Context(), nbg.LanguageEnglish, nbg.RateSetsQuery{})
	if err != nil {
		t.Fatalf("RateSets() error = %v", err)
	}

	if len(sets) != 1 || len(sets[0].Currencies) < 40 {
		t.Fatalf("RateSets() = %+v, want one set of 40 currencies or more", sets)
	}

	for _, currency := range sets[0].Currencies {
		if currency.Code == dollar {
			return
		}
	}

	t.Error("the set holds no USD")
}
