// Package mjc is a client of the exchange rates of MJC Credit, a microfinance
// organization of Georgia.
//
// MJC publishes no API of its rates. The client reads the rate table of the page
// mjc.ge/rates, a page for people. The page can change with no notice. The client
// returns each row of the table as the page shows it, with no official rate. It needs no
// key, and it does not retry and does not cache.
package mjc
