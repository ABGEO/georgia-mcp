package geostat_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/geostat"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Query() {
	body, _ := os.ReadFile("testdata/cpi-monthly-query-en.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := geostat.New(vendorhttp.WithBaseURL(server.URL))

	dataset, err := client.Query(context.Background(), "en",
		"Price Statistics/CPI/National Consumer Price Index/"+
			"CPI_in_Georgia_(previous_month=100)_orenovani2.px",
		[]geostat.Selection{
			{Code: "Region", Values: []string{"0"}},
			{Code: "Groups", Values: []string{"0", "1"}},
			{Code: "Year", Values: []string{"22"}},
			{Code: "month", Values: []string{"9"}},
		})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, datum := range dataset.Data {
		fmt.Println(datum.Key, datum.Values[0])
	}
	// Output:
	// [0 0 22 9] 100.9
	// [0 1 22 9] 100.0
}
