package rico_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/rico"
)

func ExampleClient_Rates() {
	body, _ := os.ReadFile(recordedPage)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := rico.New(rico.WithBaseURL(server.URL))

	rows, err := client.Rates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	row := rows[0]
	fmt.Println(row.Unit, row.Currency+": buy", row.Buy, "sell", row.Sell, "GEL")
	// Output: 1 USD: buy 2.5960 sell 2.6000 GEL
}
