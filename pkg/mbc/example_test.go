package mbc_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/mbc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_CommercialRates() {
	body, _ := os.ReadFile("testdata/commercial.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := mbc.New(vendorhttp.WithBaseURL(server.URL))

	rates, err := client.CommercialRates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, rate := range rates {
		if rate.FromCcy == "USD" {
			fmt.Println(
				rate.Multiplier,
				rate.FromCcy+": buy",
				rate.Buy,
				"sell",
				rate.Sell,
				rate.ToCcy,
			)
		}
	}
	// Output: 1 USD: buy 2.5950 sell 2.5990 GEL
}
