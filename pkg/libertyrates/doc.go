// Package libertyrates is a client of the exchange rates of Liberty Bank.
//
// Liberty Bank publishes no API of its rates. The client reads the rate block of the
// Georgian home page www.libertybank.ge/ka/, a page for people. The page can change
// with no notice. The client returns each currency of the block as the page shows it.
// It needs no key, and it does not retry and does not cache.
package libertyrates
