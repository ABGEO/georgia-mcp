package tas_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/tas"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_SearchDocuments() {
	body, _ := os.ReadFile("testdata/search-number.js")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := tas.New(vendorhttp.WithBaseURL(server.URL))

	page, err := client.SearchDocuments(context.Background(), tas.Filter{
		DocumentNo: "AR11168188",
		StatusIDs:  []int{1, 2},
		Limit:      50,
	})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, document := range page.Documents {
		fmt.Println(document.DocumentNo, document.DocumentStatusID)
	}
	// Output:
	// AR11168188 1
}
