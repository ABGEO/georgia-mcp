//go:build live

package geostat_test

import (
	"slices"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/geostat"
)

const annualPath = "Price Statistics/CPI/National Consumer Price Index/" +
	"CPI_in_Georgia(same_month_of_the_previous_year=100_orenovani2.px"

// GEOSTAT-SC-026: The table of salaries has the dimension Region, and the annual index of
// 2026 has a figure for Georgia, the group Total, and January.
func TestLiveTables(t *testing.T) {
	t.Parallel()

	client := geostat.New()

	wages, err := client.Table(t.Context(), "en", "Social Statistics/Labour/Wages_Region.px")
	if err != nil {
		t.Fatalf("Table() error = %v", err)
	}

	if wages.Variables[0].Code != "Region" {
		t.Errorf("first dimension = %q, want Region", wages.Variables[0].Code)
	}

	annual, err := client.Table(t.Context(), "en", annualPath)
	if err != nil {
		t.Fatalf("Table() error = %v", err)
	}

	year := annual.Variables[2]
	code := year.Values[slices.Index(year.ValueTexts, "2026")]

	dataset, err := client.Query(t.Context(), "en", annualPath, []geostat.Selection{
		{Code: "Region", Values: []string{"0"}},
		{Code: "Groups", Values: []string{"0"}},
		{Code: "Year", Values: []string{code}},
		{Code: annual.Variables[3].Code, Values: []string{"0"}},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(dataset.Data) != 1 || dataset.Data[0].Values[0] == ".." {
		t.Errorf("Query() = %+v, want one published figure", dataset.Data)
	}
}
