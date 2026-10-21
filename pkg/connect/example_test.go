package connect_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/connect"
)

func ExampleClient_Current() {
	body, _ := os.ReadFile("testdata/current.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := connect.New(connect.WithBaseURL(server.URL))

	snapshot, err := client.Current(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	price := snapshot.Prices["ron96"]
	fmt.Println(price.Label+":", price.SelfService, "GEL at a self-service station")
	// Output: პრემიუმი RON 96: 3.95000000000000017763568394002504646778106689453125 GEL at a self-service station
}
