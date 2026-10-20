package tbcoffers

// Filter is one filter of the offers, or a group of filters: Category, ProductType,
// OfferType, or CardType. Name is in the locale of the request.
type Filter struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	ItemCount int      `json:"itemCount"`
	Childs    []Filter `json:"childs"`
}

// OffersRequest selects one page of offers.
type OffersRequest struct {
	// Filter holds filters as "<group>:<code>": "Category:Leisure". Empty gives every offer.
	Filter []string `json:"filter"`
	// Locale is "ka-GE" or "en-US".
	Locale string `json:"locale"`
	// Segment is a code of a segment: "Concept". Empty gives every segment.
	Segment   string `json:"segment,omitempty"`
	PageIndex int    `json:"pageIndex"`
	PageSize  int    `json:"pageSize"`
}

// OffersPage is one page of offers.
type OffersPage struct {
	PagingDetails PagingDetails `json:"pagingDetails"`
	List          []Entry       `json:"list"`
}

// PagingDetails gives the count of every offer and whether a next page exists.
type PagingDetails struct {
	TotalCount  int  `json:"totalCount"`
	HasNextPage bool `json:"hasNextPage"`
}

// Entry is one offer of TBC Bank.
type Entry struct {
	ID          string `json:"$id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
	// StartDate and EndDate have no zone: "2026-10-20T00:00:00". Empty when absent.
	StartDate string    `json:"startDate"`
	EndDate   string    `json:"endDate"`
	Segments  []Segment `json:"segments"`
	Partner   *Partner  `json:"partner"`
	// Content holds the terms. Only the page of one offer holds it.
	Content *Node `json:"content"`
}

// Segment is a group of customers of an offer. Label is in the locale of the request.
type Segment struct {
	ID       string `json:"$id"`
	Label    string `json:"label"`
	IsHidden bool   `json:"isHidden"`
}

// Partner is the company that gives an offer.
type Partner struct {
	Title string `json:"title"`
}

// Node is one node of the tree of the terms: document, paragraph, unordered-list,
// list-item, or text. A text node holds its Value.
type Node struct {
	NodeType string `json:"nodeType"`
	Value    string `json:"value"`
	Content  []Node `json:"content"`
}
