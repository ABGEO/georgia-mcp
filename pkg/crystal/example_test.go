package crystal_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/crystal"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_CashRates() {
	body, _ := os.ReadFile("testdata/web-rate-cash.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := crystal.New(vendorhttp.WithBaseURL(server.URL))

	rates, err := client.CashRates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	rate := rates[0]
	fmt.Println(
		rate.Multiplier,
		rate.SourceISO+": buy",
		rate.BuyRate,
		"sell",
		rate.SellRate,
		rate.TargetISO,
	)
	// Output: 1 USD: buy 2.595 sell 2.599 GEL
}
