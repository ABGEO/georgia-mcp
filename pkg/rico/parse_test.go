package rico_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/rico"
)

// RICORATE-SC-003: The parse of any page returns rows or an error, and never panics.
func FuzzParseRates(f *testing.F) {
	page, err := os.ReadFile(recordedPage)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(page)
	f.Add([]byte(`<div class="currencies-table-body"><div class="currency-row">` +
		`<div class="currency-info"><strong>USD</strong></div></div></div>`))
	f.Add([]byte(`<div class="currencies-table-body">`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		rows, err := rico.ParseRates(body)
		if err != nil && rows != nil {
			t.Errorf("ParseRates() = %v and error %v, want one of the two", rows, err)
		}
	})
}
