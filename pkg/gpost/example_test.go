package gpost_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/gpost"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_PostalCodes() {
	page, _ := os.ReadFile("testdata/postal-codes.html")
	result, _ := os.ReadFile("testdata/find-rustavi-en.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Set-Cookie", "__RequestVerificationToken=test-token; path=/; HttpOnly")
			_, _ = w.Write(page)

			return
		}

		_, _ = w.Write(result)
	}))

	defer server.Close()

	client := gpost.New(vendorhttp.WithBaseURL(server.URL))

	rows, err := client.PostalCodes(context.Background(), "Rustavi", "en")
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, row := range rows[:2] {
		fmt.Println(row.Code, row.Address+",", row.District)
	}
	// Output:
	// 3700 City Rustavi, Rustavi City
	// 0500 Village Rustavi, Aspindzis District
}
