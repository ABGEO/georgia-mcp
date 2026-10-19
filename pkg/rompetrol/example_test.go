package rompetrol_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/rompetrol"
)

func ExampleClient_Prices() {
	body, _ := os.ReadFile("testdata/home.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := rompetrol.New(rompetrol.WithBaseURL(server.URL))

	rows, err := client.Prices(context.Background())
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, row := range rows[:2] {
		fmt.Println(row.Product+":", row.Price, "GEL")
	}
	// Output:
	// efix სუპერი: 4.65 GEL
	// - efix სუპერი (თვითმომსახურების სადგურებზე): 4.55 GEL
}
