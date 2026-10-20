package bogoffers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/bogoffers"
)

func ExampleClient_Offers() {
	body, _ := os.ReadFile("testdata/offers-en-0.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := bogoffers.New(bogoffers.WithBaseURL(server.URL))

	page, err := client.Offers(
		context.Background(),
		bogoffers.OffersQuery{Language: "en", PageNumber: 0, PageSize: 100},
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, offer := range page.Result.Offers {
		if offer.CampaignID == 17409 {
			fmt.Println(offer.Title+":", strings.TrimSpace(offer.ShortDesc))
		}
	}
	// Output: Ronny's Pizza: 20% Discount, unlimited
}
