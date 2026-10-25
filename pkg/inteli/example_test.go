package inteli_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/inteli"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Rates() {
	body, _ := os.ReadFile(recordedPage)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := inteli.New(vendorhttp.WithBaseURL(server.URL))

	rows, err := client.Rates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	row := rows[0]
	fmt.Println(row.Currency+": buy", row.Buy, "sell", row.Sell, row.Base)
	// Output: USD: buy 2.595 sell 2.598 GEL
}
