package cavea_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/cavea"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_MovieSessions() {
	body, _ := os.ReadFile("testdata/sessions-all-street-fighter.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := cavea.New(vendorhttp.WithBaseURL(server.URL))

	sessions, err := client.MovieSessions(context.Background(), "6ab27d3fde5280679e341878")
	if err != nil {
		fmt.Println(err)

		return
	}

	first := sessions[0]
	fmt.Println(first.ID, first.SessionLanguage, "VIP:", first.Prices["vip"], "GEL")
	// Output: cag_5/27-10-2026/22:15 geo VIP: 29 GEL
}
