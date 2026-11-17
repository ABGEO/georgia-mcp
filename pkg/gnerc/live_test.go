//go:build live

package gnerc_test

import (
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/gnerc"
)

// GNERCMON-SC-027: The default client reads the links of data.gnerc.org and the workbook
// of the balance, which holds a sheet of this year or the last.
func TestLiveBalance(t *testing.T) {
	t.Parallel()

	client := gnerc.New()

	links, err := client.Links(t.Context())
	if err != nil {
		t.Fatalf("Links() error = %v", err)
	}

	var name string

	for _, link := range links {
		address, err := url.Parse(link.Href)
		if err == nil && strings.Contains(path.Base(address.Path), "ფაქტიური ბალანსი") {
			name = path.Base(address.Path)
		}
	}

	if name == "" {
		t.Fatalf("Links() = %+v, want the link of the balance", links)
	}

	book, err := client.Workbook(t.Context(), name)
	if err != nil {
		t.Fatalf("Workbook() error = %v", err)
	}

	year := time.Now().Year()
	for _, sheet := range book.Sheets {
		if sheet.Name == strconv.Itoa(year)+" GEO" || sheet.Name == strconv.Itoa(year-1)+" GEO" {
			return
		}
	}

	t.Errorf("Workbook() has no sheet of %d or %d", year, year-1)
}
