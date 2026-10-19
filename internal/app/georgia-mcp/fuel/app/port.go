package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
)

// PriceList is every active product of one fuel company, and the facts of its read:
// the instant the server read it from the company, and whether it is a list read
// before an outage of the company.
type PriceList struct {
	Prices []domain.ProductPrice
	ReadAt time.Time
	Stale  bool
}

// PriceSource reads the current prices of one fuel company.
type PriceSource interface {
	Company() domain.Company
	Prices(ctx context.Context) (PriceList, error)
}
