//go:build live

package nager_test

import (
	"slices"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/nager"
)

// HOLIDAY-SC-022: The holidays of Georgia in the current year are at least 15, each of
// the type Public.
func TestLivePublicHolidays(t *testing.T) {
	t.Parallel()

	holidays, err := nager.New().PublicHolidays(t.Context(), time.Now().Year(), "GE")
	if err != nil {
		t.Fatalf("PublicHolidays() error = %v", err)
	}

	if len(holidays) < 15 {
		t.Errorf("len(holidays) = %d, want at least 15", len(holidays))
	}

	for _, holiday := range holidays {
		if !slices.Contains(holiday.Types, "Public") {
			t.Errorf("%s has the types %v, want Public", holiday.Date, holiday.Types)
		}
	}
}
