package mjc_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/mjc"
)

// MJCRATE-SC-003: The parse of any page returns rows or an error, and never panics.
func FuzzParseRates(f *testing.F) {
	page, err := os.ReadFile(recordedPage)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(page)
	f.Add([]byte(`<table><thead><tr><th>ვალუტა</th><th>ყიდვა</th><th>გაყიდვა</th></tr></thead>` +
		`<tr><td>1 USD</td></tr></table>`))
	f.Add([]byte(`<table>`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		rows, err := mjc.ParseRates(body)
		if err != nil && rows != nil {
			t.Errorf("ParseRates() = %v and error %v, want one of the two", rows, err)
		}
	})
}
