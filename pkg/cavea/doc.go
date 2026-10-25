// Package cavea is a client of the cinemas, movies, sessions, and seats of Cavea, the
// largest chain of cinemas of Georgia.
//
// Cavea publishes no API. The client reads the JSON methods under /api/ that the pages
// of cavea.ge call, and the data stream of the pages /movies and /sessions. The source
// is unofficial, and it can change with no notice. The client needs no key, and it does
// not retry and does not cache.
package cavea
