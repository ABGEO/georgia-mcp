package domain

import (
	"fmt"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxTableCodeRunes = 300
	tableExtension    = ".px"
)

// TableCode is the folder path and the file name of a table of Geostat:
// "Social Statistics/Labour/Wages_Region.px".
type TableCode struct {
	path string
}

// ParseTableCode returns the code of a text of 1 to 300 runes. The segments split by "/"
// are not empty, not "." and not "..", and the last ends with ".px" in any case.
func ParseTableCode(text string) (TableCode, error) {
	if text == "" || utf8.RuneCountInString(text) > maxTableCodeRunes ||
		len(text) < len(tableExtension) ||
		!strings.EqualFold(text[len(text)-len(tableExtension):], tableExtension) {
		return TableCode{}, fmt.Errorf("%w: %q", ErrInvalidTableCode, text)
	}

	for segment := range strings.SplitSeq(text, "/") {
		if segment == "" || segment == "." || segment == ".." ||
			strings.EqualFold(segment, tableExtension) {
			return TableCode{}, fmt.Errorf("%w: %q", ErrInvalidTableCode, text)
		}
	}

	return TableCode{path: text}, nil
}

// String returns the code: "Social Statistics/Labour/Wages_Region.px".
func (c TableCode) String() string {
	return c.path
}

// Folder returns the path of the folder of the table: "Social Statistics/Labour". A table
// at the root gives "".
func (c TableCode) Folder() string {
	folder := path.Dir(c.path)
	if folder == "." {
		return ""
	}

	return folder
}

// File returns the file name of the table: "Wages_Region.px".
func (c TableCode) File() string {
	return path.Base(c.path)
}

// CatalogEntry is a read model: one table of the catalog of Geostat.
type CatalogEntry struct {
	Code  TableCode
	Title string
	// Folder holds the names of the folders, from the subject down:
	// "Social Statistics / Labour".
	Folder string
	// Updated is the instant at which Geostat last changed the table, in
	// kernel.TbilisiZone().
	Updated time.Time
}

// DimensionValue is one value of a dimension: the code "1" and the text "Tbilisi".
type DimensionValue struct {
	Code string
	Text string
}

// Dimension is one axis of a table, with its values in the order of Geostat.
type Dimension struct {
	Code   string
	Text   string
	Values []DimensionValue
}

// Table is a read model: a table of the catalog with its dimensions.
type Table struct {
	Entry      CatalogEntry
	Dimensions []Dimension
}
