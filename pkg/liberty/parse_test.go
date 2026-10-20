package liberty_test

import (
	"os"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/liberty"
)

// LIBOFFER-SC-003: No page panics the parse of the page of an offer.
func FuzzParsePage(f *testing.F) {
	page, err := os.ReadFile("testdata/details-727.html")
	if err != nil {
		f.Fatal(err)
	}

	for _, seed := range [][]byte{page, []byte(""), []byte("<div class=\"featured-hero__text\"><p>a"), []byte("<<p>>")} {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, body []byte) {
		_, _ = liberty.ParsePage(body)
	})
}
