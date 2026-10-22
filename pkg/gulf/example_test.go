package gulf_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/gulf"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_PriceArchive() {
	body, _ := os.ReadFile("testdata/fuel-prices.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := gulf.New(vendorhttp.WithBaseURL(server.URL))

	table, err := client.PriceArchive(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	newest := table.Rows[0]
	fmt.Println(table.Columns[5].Name+":", newest[5], "GEL from", newest[0])
	// Output: Euro Diesel: 4.99 GEL from 2026-10-19
}
