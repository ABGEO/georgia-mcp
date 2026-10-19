// Package rompetrol is a client of the fuel prices of Rompetrol Georgia.
//
// Rompetrol publishes no API. The client reads the price table of the home page
// www.rompetrol.ge, a page for people. The page can change with no notice. The client
// returns each row of the table as the page shows it. It needs no key, and it does not
// retry and does not cache.
package rompetrol
