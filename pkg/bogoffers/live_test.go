//go:build live

package bogoffers_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/bogoffers"
)

// BOGOFFER-SC-003: The real BOG answers one offer or more, each with a campaign id and a title.
func TestLiveOffers(t *testing.T) {
	t.Parallel()

	page, err := bogoffers.New().
		Offers(t.Context(), bogoffers.OffersQuery{Language: "en", PageSize: 100})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if page.Code != 0 || len(page.Result.Offers) == 0 {
		t.Fatalf(
			"page = code %d, %d offers; want 0 and one offer or more",
			page.Code,
			len(page.Result.Offers),
		)
	}

	for _, offer := range page.Result.Offers {
		if offer.CampaignID == 0 || offer.Title == "" {
			t.Errorf("offer = %d %q, want a campaign id and a title", offer.CampaignID, offer.Title)
		}
	}
}
