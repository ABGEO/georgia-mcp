package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// OfferQuery asks for the terms of one offer.
type OfferQuery struct {
	Bank     domain.BankCode
	ID       string
	Language kernel.Language
}

// OfferDetail is one offer with its terms, and the facts of the read of the terms.
type OfferDetail struct {
	Bank   domain.Bank
	Offer  domain.Offer
	Terms  string
	ReadAt time.Time
	Stale  bool
}

// GetOffer returns the terms of one current offer of a bank of the server.
type GetOffer struct {
	banks banks
	now   func() time.Time
}

// NewGetOffer returns the use case over one source for each bank.
func NewGetOffer(sources []OfferSource, now func() time.Time) *GetOffer {
	return &GetOffer{banks: newBanks(sources), now: now}
}

// Get finds the offer in the catalog of its bank. When the catalog holds the terms, Get
// returns them with the facts of the catalog and sends no call. Else it reads the terms
// from the source.
func (u *GetOffer) Get(ctx context.Context, query OfferQuery) (OfferDetail, error) {
	bank, err := u.banks.find(query.Bank)
	if err != nil {
		return OfferDetail{}, err
	}

	source := u.banks.sources[bank.Code]

	catalog, err := source.Offers(ctx, query.Language)
	if err != nil {
		return OfferDetail{}, fmt.Errorf("reading the offers of %s: %w", bank.Name, err)
	}

	today := kernel.Today(u.now())

	index := slices.IndexFunc(catalog.Offers, func(offer domain.Offer) bool {
		return offer.ID == query.ID && !offer.Period.EndsBefore(today)
	})
	if index < 0 {
		return OfferDetail{}, fmt.Errorf("%w: %s %q", ErrOfferNotFound, bank.Name, query.ID)
	}

	offer := catalog.Offers[index]
	detail := OfferDetail{Bank: bank, Offer: offer}

	if offer.Terms != "" {
		detail.Terms, detail.ReadAt, detail.Stale = offer.Terms, catalog.ReadAt, catalog.Stale

		return detail, nil
	}

	terms, err := source.Terms(ctx, offer, query.Language)
	if err != nil {
		return OfferDetail{}, fmt.Errorf("reading the terms of %s %q: %w", bank.Name, offer.ID, err)
	}

	detail.Terms, detail.ReadAt, detail.Stale = terms.Text, terms.ReadAt, terms.Stale

	return detail, nil
}
