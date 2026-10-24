package gpost_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/gpost"
)

// POSTCODE-SC-007: The parse of any answer returns rows or an error, and never panics.
func FuzzParseRows(f *testing.F) {
	for _, path := range []string{
		"testdata/find-chavchavadze-ka.html",
		"testdata/find-none-ka.html",
		"testdata/find-malformed.html",
	} {
		recorded, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
		if err != nil {
			f.Fatal(err)
		}

		f.Add(recorded)
	}

	f.Add([]byte(`<div class="com-district">x</div><div class="com-cell-postcode">`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		rows, err := gpost.ParseRows(body)
		if err != nil && rows != nil {
			t.Errorf("ParseRows() = %v and error %v, want one of the two", rows, err)
		}

		for _, row := range rows {
			if row.District == "" || row.Code == "" || row.Address == "" {
				t.Errorf("row = %q, want a district, a code, and an address", row)
			}
		}
	})
}
