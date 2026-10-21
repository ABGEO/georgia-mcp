package kursi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/kursi"
)

func ExampleClient_Currencies() {
	body, _ := os.ReadFile("testdata/currencies.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := kursi.New(kursi.WithBaseURL(server.URL))

	pairs, err := client.Currencies(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	pair := pairs[0]
	fmt.Println(
		"1",
		pair.SecondaryCurrencyCode+": buy",
		pair.BuyRate,
		"sell",
		pair.SellRate,
		pair.BaseCurrencyCode,
	)
	// Output: 1 USD: buy 2.5940 sell 2.6010 GEL
}
