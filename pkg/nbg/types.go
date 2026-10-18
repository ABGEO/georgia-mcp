package nbg

import (
	"encoding/json"
	"encoding/xml"
)

// RateSet is the answer of the NBG for one date.
type RateSet struct {
	// Date is the valid date of the set, as midnight with the suffix Z.
	Date       string     `json:"date"`
	Currencies []Currency `json:"currencies"`
}

// Currency is the official rate of one currency. The field names keep the spelling
// of the NBG.
type Currency struct {
	Code     string `json:"code"`
	Quantity int    `json:"quantity"`
	// RateFormated holds the rate with the four decimals that the NBG publishes.
	RateFormated string `json:"rateFormated"`
	// DiffFormated holds the change with four decimals and no sign.
	DiffFormated string      `json:"diffFormated"`
	Rate         json.Number `json:"rate"`
	Name         string      `json:"name"`
	// Diff holds the signed change.
	Diff json.Number `json:"diff"`
	// Date is the publication instant. It carries the suffix Z, but the NBG writes
	// the wall time of Tbilisi.
	Date          string `json:"date"`
	ValidFromDate string `json:"validFromDate"`
}

// Code is a currency that the NBG knows.
type Code struct {
	Code string `json:"code"`
	// Name is in Georgian. The NBG ignores a language for the codes.
	Name string `json:"name"`
}

// RSS is the RSS 2.0 feed of the rates.
type RSS struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel is the channel of the feed.
type RSSChannel struct {
	Title          string    `xml:"title"`
	Link           string    `xml:"link"`
	Description    string    `xml:"description"`
	Language       string    `xml:"language"`
	Copyright      string    `xml:"copyright"`
	PubDate        string    `xml:"pubDate"`
	LastBuildDate  string    `xml:"lastBuildDate"`
	ManagingEditor string    `xml:"managingEditor"`
	WebMaster      string    `xml:"webMaster"`
	Items          []RSSItem `xml:"item"`
}

// RSSItem is one rate set of the feed.
type RSSItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
	// Description holds the rates as the HTML table of the NBG.
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}
