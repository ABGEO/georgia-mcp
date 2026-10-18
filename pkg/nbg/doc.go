// Package nbg is a client of the official exchange rates that the National Bank of
// Georgia publishes.
//
// It covers every address of the API: the rates as JSON, the RSS feed, and the codes.
// The NBG links the JSON and the RSS address from its page of rates, and publishes no
// documentation of the API. The client mirrors the answer of the NBG. It does not retry
// and does not cache.
package nbg
