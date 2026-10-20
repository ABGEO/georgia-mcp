//go:build live

package tbcoffers_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
)

// TBCOFFER-SC-003: The real TBC answers one entry or more, each with an id and a title.
func TestLiveOffers(t *testing.T) {
	t.Parallel()

	page, err := tbcoffers.New().
		Offers(t.Context(), tbcoffers.OffersRequest{Locale: "en-US", PageSize: 1000})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if len(page.List) == 0 {
		t.Fatal("Offers() holds no entry")
	}

	for _, entry := range page.List {
		if entry.ID == "" || entry.Title == "" {
			t.Errorf("entry = %q %q, want an id and a title", entry.ID, entry.Title)
		}
	}
}
