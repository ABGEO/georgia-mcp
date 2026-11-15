//go:build live

package tas_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/tas"
)

// TASPERM-SC-024: A parcel with applications gives at least 5 of them, and a number gives
// its application.
func TestLiveSearch(t *testing.T) {
	t.Parallel()

	statuses := []int{1, 2, 5, 6, 7, 9, 11, 12, 20, 21, 25, 26, 27, 28}
	client := tas.New()

	page, err := client.SearchDocuments(t.Context(), tas.Filter{
		CadastralCode: freedomCode,
		StatusIDs:     statuses,
		Limit:         500,
	})
	if err != nil || len(page.Documents) < 5 {
		t.Errorf("SearchDocuments(code) = %d, %v, want at least 5", len(page.Documents), err)
	}

	page, err = client.SearchDocuments(t.Context(), tas.Filter{
		DocumentNo: approvedNo,
		StatusIDs:  statuses,
		Limit:      50,
	})
	if err != nil || len(page.Documents) == 0 || page.Documents[0].DocumentNo != approvedNo {
		t.Errorf("SearchDocuments(number) = %+v, %v, want %s", page, err, approvedNo)
	}
}
