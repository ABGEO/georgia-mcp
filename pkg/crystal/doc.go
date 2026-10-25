// Package crystal is a client of the exchange rates of Crystal, a microfinance
// organization of Georgia.
//
// Crystal publishes no documented API. The client calls the rate method of the rate app
// currency.crystal.ge, which the page crystal.ge/valutis-kursebi/ shows in a frame. The
// method is unofficial and can change with no notice. The script of the app sends a
// token, but the method answers without it, so the client sends none. The client does
// not retry and does not cache.
package crystal
