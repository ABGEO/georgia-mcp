package geostat

// Node is a folder or a table in the list of a folder.
type Node struct {
	// ID is the name of the folder or of the file of the table: "CPI", "Wages_Region.px".
	ID string `json:"id"`
	// Type is "l" for a folder and "t" for a table.
	Type string `json:"type"`
	Text string `json:"text"`
	// Updated is the wall time of Tbilisi, with no zone: "2026-10-08T19:17:47". A table
	// only.
	Updated string `json:"updated,omitempty"`
}

// TableMetadata is the title and the dimensions of a table.
type TableMetadata struct {
	Title     string     `json:"title"`
	Variables []Variable `json:"variables"`
}

// Variable is one dimension of a table. Values holds the codes and ValueTexts the texts,
// in the same order.
type Variable struct {
	Code       string   `json:"code"`
	Text       string   `json:"text"`
	Values     []string `json:"values"`
	ValueTexts []string `json:"valueTexts"`
}

// Selection is the value codes of one dimension of a query.
type Selection struct {
	Code   string
	Values []string
}

// Dataset is the answer of a query.
type Dataset struct {
	Columns []Column `json:"columns"`
	Data    []Datum  `json:"data"`
}

// Column is a dimension or the figure of a dataset.
type Column struct {
	Code string `json:"code"`
	Text string `json:"text"`
	// Type is "d" for a dimension and "c" for the figure.
	Type string `json:"type"`
}

// Datum is one cell: the value code of each dimension and the figure as Geostat writes
// it. Geostat writes ".." for a figure that it did not publish.
type Datum struct {
	Key    []string `json:"key"`
	Values []string `json:"values"`
}

type queryBody struct {
	Query    []querySelection `json:"query"`
	Response queryResponse    `json:"response"`
}

type querySelection struct {
	Code      string    `json:"code"`
	Selection valueList `json:"selection"`
}

type valueList struct {
	Filter string   `json:"filter"`
	Values []string `json:"values"`
}

type queryResponse struct {
	Format string `json:"format"`
}
