package liberty_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/abgeo/georgia-mcp/pkg/liberty"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Offers() {
	body, _ := os.ReadFile("testdata/offers-1.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := liberty.New(vendorhttp.WithBaseURL(server.URL))

	offers, err := client.Offers(context.Background(), liberty.OffersQuery{Page: 1, Language: "en"})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, offer := range offers {
		if offer.ID == 727 {
			fmt.Println(
				strings.TrimSpace(offer.Title)+":",
				offer.Labels[0]+",",
				offer.RemainingDays,
			)
		}
	}
	// Output: Hyatt Regency Bangkok Sukhumvit: Special Hotel Offer, 72 Days left
}
