// Package credo is a client of the exchange rates of Credo Bank, a commercial bank of
// Georgia.
//
// Credo Bank publishes no API. The client calls the rate method that the site
// credobank.ge reads, an unofficial API that Noxtton hosts. The method can change with
// no notice. It needs a key in the header x-api-key. The client holds no key: the
// caller passes one with WithAPIKey. The client does not retry and does not cache.
package credo
