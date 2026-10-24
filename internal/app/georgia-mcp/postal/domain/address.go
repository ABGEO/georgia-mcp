package domain

// Address is an address of Georgian Post with its postal code.
type Address struct {
	// Text is the settlement and the street: "ქალაქი თბილისი, ამბერკი ტაბიძის ქუჩა".
	Text string
	// District is the city or the municipality that groups the address: "თბილისი".
	District string
	Code     PostalCode
}
