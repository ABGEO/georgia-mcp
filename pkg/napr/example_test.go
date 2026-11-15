package napr_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/napr"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_ParcelsAt() {
	body, _ := os.ReadFile("testdata/near-freedom.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := napr.New(vendorhttp.WithBaseURL(server.URL))

	parcels, err := client.ParcelsAt(context.Background(), 41.6934, 44.8015)
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, parcel := range parcels[:2] {
		fmt.Println(parcel.Name, parcel.Distance)
	}
	// Output:
	// 01.15.05.066.020 0
	// 01.15.05.066.015 91.775768836483
}
