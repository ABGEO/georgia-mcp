// Package basisbank is a client of the exchange rates of Basisbank, a commercial bank of
// Georgia.
//
// Basisbank publishes no API. The client calls the rate method that the page
// bb.ge/ge/currencies reads, an unofficial API of the site. The method can change with
// no notice. The client needs no key, and it does not retry and does not cache.
package basisbank
