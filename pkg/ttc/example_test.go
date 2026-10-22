package ttc_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/ttc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_ArrivalTimes() {
	body, _ := os.ReadFile("testdata/arrival-times-385.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := ttc.New(vendorhttp.WithBaseURL(server.URL), ttc.WithAPIKey("<key>"))

	arrivals, err := client.ArrivalTimes(
		context.Background(),
		"1:1000",
		ttc.ArrivalTimesQuery{Locale: "en", RouteID: "1:R29925", Max: 2},
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	first := arrivals[0]
	fmt.Println(first.ShortName, first.Headsign, "in", first.RealtimeArrivalMinutes, "minutes")
	// Output: 385 Vake - Saburtalo in 8 minutes
}
