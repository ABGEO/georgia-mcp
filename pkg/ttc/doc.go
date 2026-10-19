// Package ttc is a client of the passenger information API of the Tbilisi Transport
// Company: the stops, the routes, the arrivals, the timetables, the trip plans, and the
// place search of Tbilisi.
//
// The TTC publishes no documentation of the API. It is unofficial: the web application
// transit.ttc.com.ge reads it. The API can change with no notice. The API needs a key in
// the header X-api-key. The web page publishes the key, and the TTC can change it. The
// client does not retry and does not cache.
package ttc
