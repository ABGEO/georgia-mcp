package nbg_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/nbg"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_RateSets() {
	body, _ := os.ReadFile("testdata/en-2026-10-17.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := nbg.New(vendorhttp.WithBaseURL(server.URL))

	sets, err := client.RateSets(
		context.Background(),
		nbg.LanguageEnglish,
		nbg.RateSetsQuery{Date: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, currency := range sets[0].Currencies {
		if currency.Code == dollar {
			fmt.Println(currency.Quantity, currency.Code, "=", currency.RateFormated, "GEL")
		}
	}
	// Output: 1 USD = 2.5982 GEL
}
