// Package halyk is a client of the exchange rates of Halyk Bank Georgia, a commercial
// bank of Georgia.
//
// Halyk Bank publishes no API of its rates. The client reads the table of the current
// rates of the Georgian page halykbank.ge/ka/individuals/currency, a page for people.
// The page can change with no notice. The client returns each row of the table as the
// page shows it. It needs no key, and it does not retry and does not cache.
package halyk
