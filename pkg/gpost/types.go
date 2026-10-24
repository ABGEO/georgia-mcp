package gpost

// PostalCodeRow is one address of the search answer, as the page shows it. Each run of
// white space in a text is one space.
type PostalCodeRow struct {
	// District is the city or the municipality that groups the address: "თბილისი", or
	// "Tbilisi City" in English.
	District string
	// Code is the postal code: "0186".
	Code string
	// Address is the settlement and the street: "ქალაქი თბილისი, ამბერკი ტაბიძის ქუჩა".
	Address string
}
