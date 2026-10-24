// Package gpost is a client of the postal codes of Georgian Post.
//
// Georgian Post publishes no API. The client reads the search of the page
// www.gpost.ge/help/postal-codes, a page for people. The page can change with no notice.
// The client returns each address of the answer as the page shows it. It needs no key,
// keeps no token between two searches, and does not retry and does not cache.
package gpost
