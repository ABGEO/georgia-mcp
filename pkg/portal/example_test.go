package portal_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/portal"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_FuelPrices() {
	body, _ := os.ReadFile("testdata/fuel-prices.rsc")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := portal.New(vendorhttp.WithBaseURL(server.URL))

	prices, err := client.FuelPrices(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, price := range prices {
		if price.FuelType.Slug == "euro-regular" {
			fmt.Println(price.Name+":", price.LatestPrice, "GEL")
		}
	}
	// Output: EURO REGULAR: 3.94 GEL
}
