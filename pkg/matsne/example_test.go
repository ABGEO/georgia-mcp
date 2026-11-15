package matsne_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/matsne"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Document() {
	body, _ := os.ReadFile("testdata/act-1043717-ka.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := matsne.New(vendorhttp.WithBaseURL(server.URL))

	doc, err := client.Document(context.Background(), "ka", "1043717", -1)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(doc.Title)
	fmt.Println("version", doc.Selected, "of", doc.Publications[0].Text)
	// Output:
	// საქართველოს საგადასახადო კოდექსი
	// version 248 of 28/10/2026
}
