package nager

// PublicHoliday is one holiday of one country.
type PublicHoliday struct {
	// Date is the day of the holiday: "2026-04-12".
	Date string `json:"date"`
	// LocalName is the name in the language of the country.
	LocalName string `json:"localName"`
	// Name is the name in English.
	Name        string   `json:"name"`
	CountryCode string   `json:"countryCode"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties"`
	LaunchYear  *int     `json:"launchYear"`
	// Types holds the kinds of the holiday: "Public", "Bank", "School", "Authorities",
	// "Optional", or "Observance".
	Types []string `json:"types"`
}
