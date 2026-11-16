package gse_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/gse"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Grid() {
	body, _ := os.ReadFile("testdata/grid-16112026-1200.xml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := gse.New(vendorhttp.WithBaseURL(server.URL))
	at := time.Date(2026, time.November, 16, 12, 0, 0, 0, time.FixedZone("Tbilisi", 4*60*60))

	grid, err := client.Grid(context.Background(), at)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, block := range grid.Blocks[5:] {
		for _, entry := range block.Entries {
			fmt.Println(strings.TrimSpace(block.Name), entry.B3, entry.Value, entry.Quality)
		}
	}
	// Output:
	// TOTAL AZERBAIJAN TIE LINES Azerbaij 20.631090 ACT
	// TOTAL ARMENIA TIE LINES Armenia 0.000000 SUB
	// TOTAL RUSSIA TIE LINES Russia 0.000000 ACT
	// TOTAL RUSSIA TIE LINES Salkhino -0.000000 ACT
	// TOTAL RUSSIA TIE LINES Java 18.299999 BLS
	// TOTAL RUSSIA TIE LINES Nakaduli 14.191387 ACT
	// TOTAL TURKEY TIE LINES Turkey 279.235687 ACT
	// TOTAL CONSUMPTION   1843.806030 NRE
}
