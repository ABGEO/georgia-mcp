package bog_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/bog"
)

func ExampleClient_CommercialRate() {
	body, _ := os.ReadFile("testdata/commercial-usd.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := bog.New(bog.WithBaseURL(server.URL))

	rate, err := client.CommercialRate(context.Background(), dollar)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println("1 USD: buy", rate.Buy, "sell", rate.Sell, "GEL")
	// Output: 1 USD: buy 2.56500000 sell 2.62900000 GEL
}
