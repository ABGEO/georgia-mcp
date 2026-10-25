package halyk_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/halyk"
)

// HALRATE-SC-003: The parse of any page returns rows or an error, and never panics.
func FuzzParseRates(f *testing.F) {
	page, err := os.ReadFile(recordedPage)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(page)
	f.Add([]byte(`<div id="currentexchangerate"><table><thead><tr><th>ვალუტა</th></tr></thead>` +
		`<tbody><tr><td>USD</td></tr></tbody></table></div>`))
	f.Add([]byte(`<div id="currentexchangerate">`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		rows, err := halyk.ParseRates(body)
		if err != nil && rows != nil {
			t.Errorf("ParseRates() = %v and error %v, want one of the two", rows, err)
		}
	})
}
