package neogas_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/neogas"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_GasPrices() {
	body, _ := os.ReadFile("testdata/home.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := neogas.New(vendorhttp.WithBaseURL(server.URL))

	prices, err := client.GasPrices(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println("CNG:", prices[0], "GEL")
	// Output: CNG: 1.60 GEL
}
