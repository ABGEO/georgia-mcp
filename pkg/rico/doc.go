// Package rico is a client of the exchange rates of Rico Credit, a microfinance
// organization of Georgia.
//
// Rico Credit publishes no API of its rates. The client reads the rate table of the
// Georgian home page www.rico.ge/ka/, a page for people. The page can change with no
// notice. The client returns each row of the table as the page shows it. It needs no
// key, and it does not retry and does not cache.
package rico
