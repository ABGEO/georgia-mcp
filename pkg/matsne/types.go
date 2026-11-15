package matsne

// SearchQuery holds the parameters of the search page. The client sends no empty field.
type SearchQuery struct {
	// Query is the text to find: "შრომის კოდექსი".
	Query string
	// QueryTarget is "title", "text", or "all".
	QueryTarget string
	// Type is "main" for the consolidated acts, or "all".
	Type string
	// Group is the identifier of a document type: "1000003" for a law.
	Group string
	// Status is "active", "expired", or "toBeActivated".
	Status string
	// Issuer is the name of the body that adopted the act.
	Issuer string
	// Number is the number of the act.
	Number string
	// SigningFrom and SigningTo bound the signing date, as "dd/mm/yyyy".
	SigningFrom, SigningTo string
	// RegistrationCode is the registration code of the act: "200000000.05.001.016012".
	RegistrationCode string
	// Sort is "signingDate_desc", "signingDate_asc", or another order of the page.
	Sort string
	// Page is the page of 20 results, from 1. Zero asks the first page.
	Page int
}

// SearchPage is one page of results, as the page shows it.
type SearchPage struct {
	// Total is the count of matched acts: the digits of "სულ მოიძებნა: 29 ( ... )" or of
	// "Search count: 1 ( ... )".
	Total string
	// Items holds each result, in the order of the page.
	Items []SearchItem
}

// SearchItem is one result of the search page.
type SearchItem struct {
	// ID is the identifier of the act, from its link: "1043717".
	ID string
	// Title is the text of the link.
	Title string
	// Class is the class attribute of the panel. "panel-success" marks an active act and
	// "panel-danger" an expired one.
	Class string
	// Fields holds the text of each small element of the panel body that holds no link,
	// with no "●": the type, the issuer, the number, and the signing date. An act with no
	// number has three fields.
	Fields []string
}

// Suggestion is one title of the method that suggests titles.
type Suggestion struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// State is "active", "expired", or "toBeActivated".
	State string `json:"state"`
}

// Document is the page of one version of an act.
type Document struct {
	// Title is the head cell of the metadata table.
	Title string
	// Meta holds each row of the metadata table with two cells, in the order of the page.
	Meta []MetaRow
	// Publications holds each consolidated version of the select #publication-switcher,
	// in the order of the page: the newest first.
	Publications []Publication
	// Selected is the value of the selected version: the version of this text.
	Selected string
	// Paragraphs holds each p of #maindoc, in the order of the page. A version whose
	// translation is underway has none.
	Paragraphs []Paragraph
}

// MetaRow is one row of the metadata table.
type MetaRow struct {
	Label, Value string
}

// Publication is one consolidated version: the value "248" and the text "28/10/2026".
type Publication struct {
	Value, Text string
}

// Paragraph is one p of the text of an act.
type Paragraph struct {
	// Class is the class attribute: "muxlixml" for an article of a Georgian page,
	// "MsoNormal" for each paragraph of an English page.
	Class string
	// Text is the text with each run of white space as one space. A sup gives "^" and its
	// text, with no space before it: "მუხლი 133^3".
	Text string
	// Bold tells that each letter of the text sits inside b or strong.
	Bold bool
}
