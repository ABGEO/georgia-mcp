package lukoil_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/lukoil"
)

func ExampleClient_PriceHistory() {
	body, _ := os.ReadFile("testdata/prices-history.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := lukoil.New(lukoil.WithBaseURL(server.URL))

	table, err := client.PriceHistory(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	last := table.Rows[len(table.Rows)-1]
	fmt.Println(table.Header[5]+":", last[5], "GEL from", last[6])
	// Output: Euro Diesel: 4.97 GEL from 2026-10-05 10:01:46
}
