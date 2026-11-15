package parliament_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/abgeo/georgia-mcp/pkg/parliament"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func ExampleClient_VoteSessions() {
	body, _ := os.ReadFile("testdata/vote-sessions-114284.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))

	defer server.Close()

	client := parliament.New(vendorhttp.WithBaseURL(server.URL))

	votes, err := client.VoteSessions(context.Background(), parliament.VoteSessionQuery{
		Limit: 10, BillReviewID: 114284,
	})
	if err != nil {
		fmt.Println(err)

		return
	}

	for _, vote := range votes.List {
		result := vote.VotingResultSet
		fmt.Println(vote.ID, vote.ImportConferenceDate, vote.BillReviewTypeRd,
			result.Yes, result.No, result.WithoutInteraction)
	}
	// Output:
	// 31510 11-11-2026 I მოსმენა 77 0 24
}
