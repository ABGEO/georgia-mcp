// Package portal is a client of the fuel prices of Portal, a fuel company of Georgia.
//
// Portal publishes no API. The page portal.com.ge/ka/fuel-prices renders on the server,
// and sends its data as a stream of JSON to the script of the page. The client asks for
// that stream and reads the price list from it. It is the data of a web page for
// people, unofficial, and it can change with no notice. The client needs no key, and it
// does not retry and does not cache.
package portal
