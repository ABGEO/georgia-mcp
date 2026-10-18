package tbc_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/tbc"
)

func ExampleClient_CommercialRates() {
	body, _ := os.ReadFile("testdata/commercial.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := tbc.New(tbc.WithBaseURL(server.URL), tbc.WithAPIKey("your-key"))

	rates, err := client.CommercialRates(context.Background(), []string{dollar})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, rate := range rates.CommercialRatesList {
		if rate.Currency == dollar {
			fmt.Println("1 USD: buy", rate.Buy, "sell", rate.Sell, rates.Base)
		}
	}
	// Output: 1 USD: buy 2.566 sell 2.628 GEL
}
