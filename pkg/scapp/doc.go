// Package scapp is a client of the exchange rates of Scapp, a microfinance organization
// of Georgia, formerly Swiss Capital.
//
// Scapp publishes no API of its rates. The client reads the rate table of the Georgian
// page scapp.ge/ka/currency, a page for people. The page can change with no notice. The
// client returns each row of the table as the page shows it. It needs no key, and it
// does not retry and does not cache.
package scapp
