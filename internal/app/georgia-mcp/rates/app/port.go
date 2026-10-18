package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// RateSetResult is a rate set and the facts of its read: the instant the server read
// it from the vendor, and whether it is a set read before an outage of the vendor.
type RateSetResult struct {
	Set    domain.RateSet
	ReadAt time.Time
	Stale  bool
}

// OfficialRateSource reads the rate set in force on a date. For a date after the
// newest set, it returns the newest set. Before the first set, it returns an empty set.
type OfficialRateSource interface {
	RateSet(ctx context.Context, date domain.Date, language kernel.Language) (RateSetResult, error)
}
