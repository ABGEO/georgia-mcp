// Package kursi is a client of the exchange rates of Kursi.ge, an online exchange
// bureau of Georgia.
//
// The method is unofficial. Kursi.ge documents a merchant API at docs.kursi.ge, whose
// method of the rates needs the token of a merchant account. The client reads the
// public method that the page kursi.ge reads, which needs no token and is not in the
// documentation. The client mirrors the answer. It does not retry and does not cache.
package kursi
