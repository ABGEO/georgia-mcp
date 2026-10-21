package libertyrates_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/libertyrates"
)

// LIBRATE-SC-003: The parse of any page returns rows or an error, and never panics.
func FuzzParseRates(f *testing.F) {
	page, err := os.ReadFile(recordedPage)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(page)
	f.Add([]byte(`<div class="currency-rates"><div class="currency-rates__row-head">` +
		`<div class="currency-rates__row"><div class="currency-rates__item">` +
		`<span class="currency-rates__item-title">კომერციული</span></div></div></div>` +
		`<div class="currency-rates__row js-homepage__currency-item"></div></div>`))
	f.Add([]byte(`<div class="currency-rates">`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		rows, err := libertyrates.ParseRates(body)
		if err != nil && rows != nil {
			t.Errorf("ParseRates() = %v and error %v, want one of the two", rows, err)
		}
	})
}
