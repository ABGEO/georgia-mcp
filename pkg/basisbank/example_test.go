package basisbank_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/basisbank"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Rates() {
	body, _ := os.ReadFile(recordedAnswer)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := basisbank.New(vendorhttp.WithBaseURL(server.URL))

	days, err := client.Rates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	day := days[0]
	buy, sell := day.XRates.KursBuy["USD"], day.XRates.KursSell["USD"]
	fmt.Println(day.Date, "USD: buy", buy, "sell", sell, "GEL")
	// Output: 2026-10-25 USD: buy 2.5600 sell 2.6300 GEL
}
