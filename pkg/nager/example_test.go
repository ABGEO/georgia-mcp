package nager_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/nager"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_PublicHolidays() {
	body, _ := os.ReadFile("testdata/ge-2026.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := nager.New(vendorhttp.WithBaseURL(server.URL))

	holidays, err := client.PublicHolidays(context.Background(), 2026, "GE")
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, holiday := range holidays[9:12] {
		fmt.Println(holiday.Date, holiday.Name)
	}
	// Output:
	// 2026-04-12 Easter Sunday
	// 2026-04-13 Easter Monday
	// 2026-05-09 Day of Victory over Fascism
}
