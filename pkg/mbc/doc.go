// Package mbc is a client of the exchange rates of MBC, a microfinance organization of
// Georgia.
//
// MBC publishes no documented API. The client calls the rate method that the page
// fx.mbc.com.ge reads, an unofficial method on port 8022. The method can change with no
// notice. It needs no key. The client returns every item of the answer, also the ones
// that the page hides. It does not retry and does not cache.
package mbc
