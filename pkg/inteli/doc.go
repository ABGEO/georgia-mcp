// Package inteli is a client of the exchange rates of Inteli Express, a microfinance
// organization of Georgia.
//
// Inteli Express publishes no API of its rates. The client reads the rate table of the
// Georgian home page inteliexpress.com/ka/main-page-geo/, a page for people. The page can
// change with no notice. The table names no column, so the client returns the cells of
// each row in the order of the page. It needs no key, and it does not retry and does not
// cache.
package inteli
