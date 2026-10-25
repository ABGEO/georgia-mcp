package scapp_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/scapp"
)

// SCAPRATE-SC-003: The parse of any page returns rows or an error, and never panics.
func FuzzParseRates(f *testing.F) {
	page, err := os.ReadFile(recordedPage)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(page)
	f.Add([]byte(`<table><thead><tr><th></th><th>ყიდვა</th></tr></thead>` +
		`<tr><td>1 USD</td></tr></table>`))
	f.Add([]byte(`<table>`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		rows, err := scapp.ParseRates(body)
		if err != nil && rows != nil {
			t.Errorf("ParseRates() = %v and error %v, want one of the two", rows, err)
		}
	})
}
