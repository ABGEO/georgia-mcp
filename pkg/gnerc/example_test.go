package gnerc_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/gnerc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Workbook() {
	body, _ := os.ReadFile("testdata/balancing.xlsx")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := gnerc.New(vendorhttp.WithBaseURL(server.URL))

	book, err := client.Workbook(
		context.Background(), "საბალანსო ელექტროენერგიის ფასები 2015 - 2026 წლები.xlsx",
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, row := range book.Sheets[0].Rows[1:3] {
		fmt.Println(row.Number, row.Cells[0].Text, row.Cells[1].Text)
	}
	// Output:
	// 5 თვე 2015
	// 6 იანვარი 13.3652487
}
