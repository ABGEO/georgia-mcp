//go:build live

package matsne_test

import (
	"slices"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/matsne"
)

// MATSNE-SC-005: The real Legislative Herald accepts the default client and gives the
// page of the Tax Code, a search, and the suggested titles.
func TestLiveLegislativeHerald(t *testing.T) {
	t.Parallel()

	client := matsne.New()
	ctx := t.Context()

	doc, err := client.Document(ctx, "ka", "1043717", -1)
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if len(doc.Publications) < 249 {
		t.Errorf("Publications = %d, want 249 or more", len(doc.Publications))
	}

	page, err := client.Search(ctx, "ka", matsne.SearchQuery{
		Query: "შრომის კოდექსი", Type: "main", Group: "1000003",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(page.Items) == 0 {
		t.Error("Search() gave no item")
	}

	titles, err := client.Suggest(ctx, "საგადასახადო კოდექსი")
	if err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}

	if !slices.ContainsFunc(titles, func(s matsne.Suggestion) bool { return s.ID == "1043717" }) {
		t.Errorf("Suggest() = %d titles, want 1043717 among them", len(titles))
	}
}
