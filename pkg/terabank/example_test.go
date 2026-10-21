package terabank_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/terabank"
)

func ExampleClient_CrossRates() {
	body, _ := os.ReadFile("testdata/cross-rates.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := terabank.New(terabank.WithBaseURL(server.URL))

	rates, err := client.CrossRates(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	rate := rates[0]
	fmt.Println("1", rate.ISO+": buy", rate.Buy, "sell", rate.Sell, "GEL")
	// Output: 1 USD: buy 2.5960 sell 2.6010 GEL
}
