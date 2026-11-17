// Package gnerc is a client of the workbooks that the Georgian National Energy and Water
// Supply Regulatory Commission (GNERC) links from data.gnerc.org: the monthly balance of
// electricity and the monthly prices.
//
// The workbooks are unofficial as an API: GNERC documents no format and changes the name
// of a file each year. They need no key. The client returns the links of the home page
// and the cells of a workbook as text. It does not retry and does not cache.
package gnerc
