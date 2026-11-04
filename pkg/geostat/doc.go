// Package geostat is a client of the PxWeb API of Geostat, the National Statistics
// Office of Georgia.
//
// The API is open and needs no key. It lists the folders and the tables of the
// database, describes the dimensions of a table, and returns the cells of a selection.
// The client mirrors the answers of Geostat. It does not retry and does not cache.
package geostat
