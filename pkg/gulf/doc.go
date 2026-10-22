// Package gulf is a client of the fuel prices of Gulf Georgia, a fuel company of Georgia.
//
// Gulf publishes no API. The client reads the table of the page gulf.ge/en/fuel_prices,
// a web page for people: one row for each change of the prices, with its date, the
// newest first. A bot filter guards the site and refuses the TLS handshake of Go, so the
// default client dials with the handshake of a browser. See NewTransport. The page can
// change with no notice. The client needs no key, and it does not retry and does not
// cache.
package gulf
