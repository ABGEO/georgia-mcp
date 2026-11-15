//go:build live

package parliament_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/parliament"
)

// PARL-SC-024: The search of "ბიუჯეტ" gives more than 1000 bills, and a page of 50.
func TestLiveBills(t *testing.T) {
	t.Parallel()

	client := parliament.New()

	bills, err := client.Bills(t.Context(), parliament.BillQuery{Limit: 50, BillName: "ბიუჯეტ"})
	if err != nil {
		t.Fatalf("Bills() error = %v", err)
	}

	if bills.Total <= 1000 || len(bills.List) != 50 {
		t.Errorf("total, len = %d, %d, want more than 1000, 50", bills.Total, len(bills.List))
	}
}

// PARL-SC-024: The vote 31507 gives 101 choices, each with one of the three names.
func TestLiveVotings(t *testing.T) {
	t.Parallel()

	client := parliament.New()

	votings, err := client.Votings(t.Context(), 31507, 0, 200)
	if err != nil {
		t.Fatalf("Votings() error = %v", err)
	}

	if len(votings.List) != 101 {
		t.Fatalf("len = %d, want 101", len(votings.List))
	}

	names := map[string]bool{"დიახ": true, "არა": true, "არ ესწრება": true}

	for _, voting := range votings.List {
		if voting.VotingResult == nil || !names[voting.VotingResult.Name] {
			t.Errorf(
				"choice of %q = %+v, want one of %v",
				voting.ColleagueRd,
				voting.VotingResult,
				names,
			)
		}
	}
}
