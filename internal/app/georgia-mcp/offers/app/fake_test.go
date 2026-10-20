package app_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	bog = "bog"
	xyz = "xyz"
)

// readAt is the read instant of every fake catalog.
func readAt() time.Time {
	return time.Date(2026, 10, 20, 6, 0, 0, 0, time.UTC)
}

// now is 2026-10-20 12:00 in Tbilisi.
func now() time.Time {
	return time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
}

// fakeSource answers a fixed list of offers, or an error.
type fakeSource struct {
	bank       domain.Bank
	offers     []domain.Offer
	err        error
	termsCalls atomic.Int32
}

func newFake(code domain.BankCode, offers ...domain.Offer) *fakeSource {
	return &fakeSource{bank: domain.Bank{Code: code, Name: string(code)}, offers: offers}
}

func (f *fakeSource) Bank() domain.Bank {
	return f.bank
}

func (f *fakeSource) Cards() []domain.Card {
	return []domain.Card{{Code: "amex", Name: "Amex"}}
}

func (f *fakeSource) Offers(context.Context, kernel.Language) (app.Catalog, error) {
	if f.err != nil {
		return app.Catalog{}, f.err
	}

	return app.Catalog{Offers: f.offers, ReadAt: readAt()}, nil
}

func (f *fakeSource) Terms(
	_ context.Context,
	offer domain.Offer,
	_ kernel.Language,
) (app.Terms, error) {
	f.termsCalls.Add(1)

	return app.Terms{Text: "the terms of " + offer.ID, ReadAt: readAt()}, nil
}

func mustDate(t *testing.T, text string) kernel.Date {
	t.Helper()

	if text == "" {
		return kernel.Date{}
	}

	date, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return date
}

// offer builds a valid offer of bog. Each change sets one value.
func offer(t *testing.T, id string, changes ...func(*domain.Offer)) domain.Offer {
	t.Helper()

	draft := domain.Offer{
		Bank:       bog,
		ID:         id,
		Title:      "Offer " + id,
		Benefit:    "20% Discount, unlimited",
		Kinds:      []domain.BenefitKind{domain.KindDiscount},
		Categories: []domain.Category{domain.CategoryOther},
		Tiers:      []domain.Tier{domain.TierStandard},
		URL:        "https://bankofgeorgia.ge/en/offers-hub/details/" + id,
	}

	for _, change := range changes {
		change(&draft)
	}

	valid, err := domain.NewOffer(draft)
	if err != nil {
		t.Fatal(err)
	}

	return valid
}

func period(t *testing.T, start, end string) func(*domain.Offer) {
	t.Helper()

	return func(o *domain.Offer) {
		o.Period = domain.Period{Start: mustDate(t, start), End: mustDate(t, end)}
	}
}

func ids(offers []domain.Offer) []string {
	out := make([]string, 0, len(offers))
	for _, o := range offers {
		out = append(out, o.ID)
	}

	return out
}
