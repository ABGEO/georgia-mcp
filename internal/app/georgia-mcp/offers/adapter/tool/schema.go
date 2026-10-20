package tool

type searchOffersInput struct {
	Banks      []string `json:"banks,omitempty"      jsonschema:"Bank codes. Empty asks every bank. The description lists the codes."`
	Tiers      []string `json:"tiers,omitempty"      jsonschema:"Customer tiers: standard, premium, private, youth. An offer matches when it has one of them."`
	Categories []string `json:"categories,omitempty" jsonschema:"Offer categories: dining, leisure, travel, health_beauty, auto, transport, shopping, electronics, home, education, kids, entertainment, real_estate, other."`
	Kinds      []string `json:"kinds,omitempty"      jsonschema:"Benefit kinds: discount, cashback, points, other."`
	Cards      []string `json:"cards,omitempty"      jsonschema:"Card codes of the banks. An offer that names no card matches every card. The description lists the codes of each bank."`
	From       string   `json:"from,omitempty"       jsonschema:"The first date of the period, YYYY-MM-DD. The default is today in Tbilisi."`
	To         string   `json:"to,omitempty"         jsonschema:"The last date of the period, YYYY-MM-DD, at most 366 days after from. The default is from."`
	Query      string   `json:"query,omitempty"      jsonschema:"A text to find in the title, the partner, the benefit, or the bank category, 1 to 100 characters."`
	Language   string   `json:"language,omitempty"   jsonschema:"The language of the texts: ka or en. The default is en."`
	Limit      int      `json:"limit,omitempty"      jsonschema:"The maximum count of offers for each bank, 1 to 30. The default is 10."`
	Offset     int      `json:"offset,omitempty"     jsonschema:"The count of matching offers of each bank to skip, 0 or more. Give the next_offset of a bank to read its next page. The default is 0."`
}

type searchOutput struct {
	Meaning     string             `json:"meaning"               jsonschema:"How to read the offers."`
	Banks       []bankOffersOutput `json:"banks"                 jsonschema:"The offers of each bank that answered, sorted by bank code."`
	Unavailable []bankOutput       `json:"unavailable,omitempty" jsonschema:"The banks that did not answer."`
}

type bankOffersOutput struct {
	Bank       string        `json:"bank"                  jsonschema:"The bank code."`
	Source     string        `json:"source"                jsonschema:"The name of the bank."`
	AsOf       string        `json:"as_of"                 jsonschema:"The instant at which the server read the offers, RFC 3339. The bank publishes no instant."`
	Stale      bool          `json:"stale"                 jsonschema:"True when the bank did not answer and the server returns offers it read before."`
	Total      int           `json:"total"                 jsonschema:"The count of every matching offer of the bank. offers holds at most limit of them."`
	NextOffset int           `json:"next_offset,omitempty" jsonschema:"The offset of the next page of the bank. Absent when offers holds the last matching offer."`
	Offers     []offerOutput `json:"offers"                jsonschema:"The matching offers, the soonest end first."`
}

type offerOutput struct {
	Bank           string       `json:"bank"                      jsonschema:"The bank code."`
	ID             string       `json:"id"                        jsonschema:"The offer id of the bank. offers_get_offer takes it."`
	Title          string       `json:"title"                     jsonschema:"The title of the offer."`
	Partner        string       `json:"partner,omitempty"         jsonschema:"The company that gives the benefit. Absent when the bank names none."`
	Benefit        string       `json:"benefit"                   jsonschema:"The text of the bank that states the benefit. Read the percent, the cap, and the conditions from it."`
	Kinds          []string     `json:"kinds"                     jsonschema:"The benefit kinds that the bank states in a field. other: the bank states none."`
	Categories     []string     `json:"categories"                jsonschema:"The offer categories, the same for every bank."`
	BankCategories []string     `json:"bank_categories,omitempty" jsonschema:"The names of the categories at the bank."`
	Tiers          []string     `json:"tiers"                     jsonschema:"The customer tiers that can use the offer."`
	Cards          []cardOutput `json:"cards,omitempty"           jsonschema:"The cards that the offer names. Absent: every card of the tiers."`
	StartDate      string       `json:"start_date,omitempty"      jsonschema:"The first date of the offer, YYYY-MM-DD. Absent when the bank publishes none."`
	EndDate        string       `json:"end_date,omitempty"        jsonschema:"The last date of the offer, YYYY-MM-DD. Absent when the bank publishes none."`
	URL            string       `json:"url"                       jsonschema:"The page of the offer at the bank. Give it to the person with each offer that you name."`
}

type cardOutput struct {
	Code string `json:"code" jsonschema:"The card code. cards of offers_search_offers takes it."`
	Name string `json:"name" jsonschema:"The name of the card at the bank."`
}

type bankOutput struct {
	Bank   string `json:"bank"   jsonschema:"The bank code."`
	Source string `json:"source" jsonschema:"The name of the bank."`
}

type getOfferInput struct {
	Bank     string `json:"bank"               jsonschema:"The bank code of the offer."`
	ID       string `json:"id"                 jsonschema:"The offer id from offers_search_offers."`
	Language string `json:"language,omitempty" jsonschema:"The language of the texts: ka or en. The default is en."`
}

type offerDetailOutput struct {
	Source string      `json:"source" jsonschema:"The name of the bank."`
	AsOf   string      `json:"as_of"  jsonschema:"The instant at which the server read the terms, RFC 3339."`
	Stale  bool        `json:"stale"  jsonschema:"True when the bank did not answer and the server returns terms it read before."`
	Offer  offerOutput `json:"offer"  jsonschema:"The offer."`
	Terms  string      `json:"terms"  jsonschema:"The full conditions of the offer, as the bank writes them."`
}
