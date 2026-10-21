// Package lukoil is a client of the price history of Lukoil Georgia, a fuel company of
// Georgia.
//
// Lukoil publishes no API. The client reads the table of the page
// www.lukoil.ge/prices-history, a web page for people: one row for each change of the
// prices, with its instant. The page can change with no notice. The client needs no key,
// and it does not retry and does not cache.
package lukoil
