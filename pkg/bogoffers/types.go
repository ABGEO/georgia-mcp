package bogoffers

import "encoding/json"

// OffersQuery selects one page of offers.
type OffersQuery struct {
	// Language is "ka" or "en", the value of the Accept-Language header. Empty sends no
	// header, and Bank of Georgia answers in Georgian.
	Language string
	// PageNumber counts from 0.
	PageNumber int
	// PageSize is 1 to 100. Bank of Georgia fails a larger page.
	PageSize int
}

// OffersPage is one page of offers.
type OffersPage struct {
	// Code is 0 on a success.
	Code   int          `json:"code"`
	Result OffersResult `json:"result"`
}

// OffersResult holds the offers of a page and the count of every offer.
type OffersResult struct {
	TotalItemCount int     `json:"totalItemCount"`
	TotalPageCount int     `json:"totalPageCount"`
	Offers         []Offer `json:"offers"`
}

// Offer is one offer of Bank of Georgia.
type Offer struct {
	CampaignID int64 `json:"campaignId"`
	// StartDate and EndDate are milliseconds of the Unix epoch. Nil when absent.
	StartDate          *int64 `json:"startDate"`
	EndDate            *int64 `json:"endDate"`
	SoloCampaign       bool   `json:"soloCampaign"`
	ActivatingCampaign bool   `json:"activatingCampaign"`
	CategoryID         int    `json:"categoryId"`
	CategoryDesc       string `json:"categoryDesc"`
	// ProductCodes and SegmentTypes hold codes separated by ";": "SOLO;VISA_CARDS".
	ProductCodes string `json:"productCodes"`
	SegmentTypes string `json:"segmentTypes"`
	// BenefitName is DISCOUNT, CASHBACK, PLUS, MR, SCOOLA, or OTHER.
	BenefitName  string      `json:"benefitName"`
	BenefitValue json.Number `json:"benefitValue"`
	BenefitBadge string      `json:"benefitBadge"`
	CityNames    string      `json:"cityNames"`
	BrandNames   string      `json:"brandNames"`
	Title        string      `json:"title"`
	// ShortDesc states the benefit: "20% Discount, unlimited".
	ShortDesc string `json:"shortDesc"`
	// LongDesc holds the terms in HTML.
	LongDesc string `json:"longDesc"`
}
