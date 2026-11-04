package nea_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/nea"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_Hourly() {
	body, _ := os.ReadFile("testdata/hourly-tsrt-week.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := nea.New(vendorhttp.WithBaseURL(server.URL))

	stations, err := client.Hourly(context.Background(), nea.HourlyQuery{
		From: "2026-10-28T00:00:00", To: "2026-11-05T23:00:00", Station: "TSRT",
	})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, equipment := range stations[0].Equipment {
		first := equipment.Values[0]
		fmt.Println(equipment.Substance.Name, first.DateTime, first.Value)
	}
	// Output:
	// PM10 2026-10-28T00:00:00 40.1208333333333
	// PM2.5 2026-10-28T03:00:00 14.2916666666667
	// NO2 2026-10-28T00:00:00 34.58
	// O3 2026-10-28T00:00:00 15.375
	// SO2 2026-10-28T00:00:00 5.225
	// CO 2026-10-28T00:00:00 0.345
}
