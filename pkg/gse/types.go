package gse

// Export is one file of GSE: the header and the blocks of values.
type Export struct {
	Header Header  `xml:"Export"`
	Blocks []Block `xml:"BLOCKS>BLOCK_AREA"`
}

// Header names the time and the period of a file.
type Header struct {
	// Time is a local time of Tbilisi: "16.11.2026 12:00:00".
	Time string `xml:"Time,attr"`
	// Period is "03M" for the grid, "01M" for the frequency.
	Period string `xml:"Period,attr"`
	Type   string `xml:"Type,attr"`
}

// Block is one group of values. GSE pads its name with spaces.
type Block struct {
	Name    string  `xml:"Name,attr"`
	Entries []Entry `xml:"TA"`
}

// Entry is one value, with the attributes of GSE as published.
type Entry struct {
	B1 string `xml:"B1,attr"`
	B2 string `xml:"B2,attr"`
	// B3 names a line: "Salkhino".
	B3      string `xml:"B3,attr"`
	Element string `xml:"EL,attr"`
	// Kind is "MvAvr03m", the average of 3 minutes, or "F" for the frequency.
	Kind    string `xml:"IN,attr"`
	Value   string `xml:"VALUE,attr"`
	Quality string `xml:"QUALITY,attr"`
}
