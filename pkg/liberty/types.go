package liberty

// Category is a category of the offers. Title is in the language of the request.
type Category struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// Type is a type of the offers: Sale, Cashback, Visa, MasterCard, or Top|Card Black
// Edition on 2026-10-20. Title is in the language of the request.
type Type struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// OffersQuery selects one page of 16 offers.
type OffersQuery struct {
	// CategoryID is 0 for every category.
	CategoryID int
	// TypeID is 0 for every type.
	TypeID int
	// Page counts from 1.
	Page int
	// Language is "ka" or "en".
	Language string
}

// Offer is one offer of Studio.
type Offer struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	ShortContent string `json:"shortContent"`
	URL          string `json:"url"`
	// Labels state the benefit: "5% Cashback".
	Labels []string `json:"labels"`
	// RemainingDays is the text of the days left: "72 Days left".
	RemainingDays string `json:"remainingDays"`
}

// Page is the text of the page of one offer, as the page shows it. Each run of white
// space is one space.
type Page struct {
	Title      string
	Timer      string
	Paragraphs []string
}
