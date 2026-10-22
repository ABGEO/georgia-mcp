package tbcoffers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Offers() {
	body, _ := os.ReadFile("testdata/offers.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := tbcoffers.New(vendorhttp.WithBaseURL(server.URL))

	page, err := client.Offers(
		context.Background(),
		tbcoffers.OffersRequest{Locale: "en-US", PageSize: 1000},
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, entry := range page.List {
		if entry.Slug == "oktopusi-qeshbeqi" {
			fmt.Println(entry.Title+":", entry.Partner.Title)
		}
	}
	// Output: Oktopus: Oktopus Fitness Club
}
