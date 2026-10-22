package libertyrates_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/libertyrates"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Rates() {
	body, _ := os.ReadFile(recordedPage)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := libertyrates.New(vendorhttp.WithBaseURL(server.URL))

	rows, err := client.Rates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	row := rows[0]
	fmt.Println(row.Unit+": buy", row.Commercial.Buy, "sell", row.Commercial.Sell, "GEL")
	// Output: 1 USD: buy 2.5470 sell 2.6470 GEL
}
