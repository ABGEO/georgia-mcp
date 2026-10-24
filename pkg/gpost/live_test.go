//go:build live

package gpost_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/gpost"
)

// POSTCODE-SC-021: The real Georgian Post gives 0186 for ამბერკი ტაბიძის ქუჩა.
func TestLivePostalCodes(t *testing.T) {
	t.Parallel()

	rows, err := gpost.New().PostalCodes(t.Context(), "ამბერკი", "ka")
	if err != nil {
		t.Fatalf("PostalCodes() error = %v", err)
	}

	for _, row := range rows {
		if row.Code == "0186" && row.Address == "ქალაქი თბილისი, ამბერკი ტაბიძის ქუჩა" {
			return
		}
	}

	t.Errorf("PostalCodes() = %q, want the row 0186 of ქალაქი თბილისი, ამბერკი ტაბიძის ქუჩა", rows)
}
