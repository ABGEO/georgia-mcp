package gnerc

// Link is one link of the home page.
type Link struct {
	// Href is the address as the page writes it, with its spaces:
	// "https://data.gnerc.org/excel/2015 - 2026 ფაქტიური ბალანსი.xlsx".
	Href string
	// Text is the text of the link with each run of spaces as one space.
	Text string
}

// Workbook is one workbook of GNERC.
type Workbook struct {
	// LastModified is the header Last-Modified of the answer, or empty.
	LastModified string
	// Sheets are in the order of the workbook.
	Sheets []Sheet
}

// Sheet is one sheet of a workbook.
type Sheet struct {
	Name string
	// Rows are the rows that hold a cell with a value, in order.
	Rows []Row
}

// Row is one row of a sheet.
type Row struct {
	// Number is 1 for the first row.
	Number int
	// Cells are the cells with a value, in order.
	Cells []Cell
}

// Cell is one cell with a value.
type Cell struct {
	// Column is the letters of the reference of the cell: "C".
	Column string
	// Type is the attribute t of the cell: "s" for a shared string, "str" for the text of
	// a formula, "inlineStr", "b", "e", or empty for a number.
	Type string
	// Text is the shared string or the inline string that the cell holds, else the text of
	// its value as the workbook writes it: "388.95957600000003".
	Text string
}
