package app

import (
	"context"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Catalog is every offer that one bank lists now, and the facts of its read: the
// instant the server read it from the bank, and whether it is a catalog read before an
// outage of the bank.
type Catalog struct {
	Offers []domain.Offer
	ReadAt time.Time
	Stale  bool
}

// Terms is the full text of the conditions of one offer, and the facts of its read.
type Terms struct {
	Text   string
	ReadAt time.Time
	Stale  bool
}

// OfferSource reads the offers of one bank.
type OfferSource interface {
	Bank() domain.Bank
	// Cards returns the cards that the source knows, sorted by code.
	Cards() []domain.Card
	Offers(ctx context.Context, language kernel.Language) (Catalog, error)
	// Terms returns the terms of an offer of its catalog whose Terms is empty.
	Terms(ctx context.Context, offer domain.Offer, language kernel.Language) (Terms, error)
}
