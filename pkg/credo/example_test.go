package credo_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/credo"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_CurrentRates() {
	body, _ := os.ReadFile("testdata/current-rates.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := credo.New(vendorhttp.WithBaseURL(server.URL), credo.WithAPIKey("your-key"))

	rates, err := client.CurrentRates(context.Background(), credo.MethodCommercial, time.Now())
	if err != nil {
		fmt.Println(err)

		return
	}

	rate := rates[0]
	fmt.Println(rate.FromCcy+": buy", rate.Buy, "sell", rate.Sell, rate.ToCcy)
	// Output: EUR: buy 2.875 sell 2.95 GEL
}
