package rompetrol_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/rompetrol"
)

// ROMFUEL-SC-003: The parse of any page returns rows or an error, and never panics.
func FuzzParsePrices(f *testing.F) {
	recorded, err := os.ReadFile("testdata/home.html")
	if err != nil {
		f.Fatal(err)
	}

	f.Add(recorded)
	f.Add([]byte(`<a id="pricelist"></a><table><tr><td>a</td><td>1</td></tr></table>`))
	f.Add([]byte(`<a id="pricelist"><table><tr><td><td><td>`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		rows, err := rompetrol.ParsePrices(body)
		if err != nil && rows != nil {
			t.Errorf("ParsePrices() = %v and error %v, want one of the two", rows, err)
		}
	})
}
