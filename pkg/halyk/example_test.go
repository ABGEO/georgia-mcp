package halyk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/halyk"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Rates() {
	body, _ := os.ReadFile(recordedPage)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := halyk.New(vendorhttp.WithBaseURL(server.URL))

	rows, err := client.Rates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	row := rows[0]
	fmt.Println(row.Currency+": buy", row.Buy, "sell", row.Sell, "GEL")
	// Output: USD: buy 2.5870 sell 2.6120 GEL
}
