//go:build live

package gulf_test

import (
	"slices"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/gulf"
)

// GULFUEL-SC-004: The real Gulf accepts the default client, gives a date column and one
// price column of the tier Gulf or more, and the first cell of each row is a date.
func TestLivePriceArchive(t *testing.T) {
	t.Parallel()

	table, err := gulf.New().PriceArchive(t.Context())
	if err != nil {
		t.Fatalf("PriceArchive() error = %v", err)
	}

	hasClass := func(class string) bool {
		return slices.ContainsFunc(
			table.Columns,
			func(c gulf.Column) bool { return c.Class == class },
		)
	}

	if !hasClass("first") || !hasClass("g_col") {
		t.Fatalf(
			"Columns = %+v, want a column of class first and one of class g_col",
			table.Columns,
		)
	}

	for _, row := range table.Rows {
		if _, err := time.Parse(time.DateOnly, row[0]); err != nil {
			t.Errorf("row %q: the first cell is not a date: %v", row, err)
		}
	}
}
