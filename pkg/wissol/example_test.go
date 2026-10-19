package wissol_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/wissol"
)

func ExampleClient_FuelPrices() {
	body, _ := os.ReadFile("testdata/fuelprice.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := wissol.New(wissol.WithBaseURL(server.URL))

	prices, err := client.FuelPrices(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, price := range prices {
		if price.FuelTypeEng == "Euro Regular" {
			fmt.Println(
				price.FuelTypeEng+":",
				price.Price,
				"GEL, self-service",
				price.PriceSelfService,
				"GEL",
			)
		}
	}
	// Output: Euro Regular: 4.08 GEL, self-service 3.93 GEL
}
