// Package matsne is a client of the Legislative Herald of Georgia (matsne.gov.ge), the
// official publisher of the legal acts of Georgia.
//
// The Legislative Herald publishes no API. The client reads its web pages for people: the
// search page, the page of an act, and the method that suggests titles. A bot filter
// guards the site and refuses the TLS handshake of Go, so the default client dials with
// the handshake of a browser. See NewTransport. The filter answers with a page of the
// status 200, and the site answers an unknown act with the same page. The pages can
// change with no notice. The client needs no key, and it does not retry and does not
// cache.
package matsne
