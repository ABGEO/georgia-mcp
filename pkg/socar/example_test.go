package socar_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/socar"
)

func ExampleClient_CurrentPrices() {
	body, _ := os.ReadFile("testdata/current-prices.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := socar.New(socar.WithBaseURL(server.URL))

	prices, err := client.CurrentPrices(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, price := range prices.Results {
		if price.FuelCode == "EURREG" {
			fmt.Println(price.FuelNameEng+":", price.FuelUnitPrice, "GEL")
		}
	}
	// Output: Nano Euro Regular: 4.02 GEL
}
