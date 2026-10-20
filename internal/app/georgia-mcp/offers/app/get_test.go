package app_test

import (
	"errors"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// BOGOFFER-SC-015: An offer with terms in the list returns them with no call to Terms.
// An offer that the bank does not list now is not found.
func TestGetOfferReturnsTheTerms(t *testing.T) {
	t.Parallel()

	source := newFake(bog,
		offer(t, "16902", func(o *domain.Offer) { o.Terms = "Mazda 3 Sedan: from 65 400 GEL" }),
		offer(t, "ended", period(t, "", "2026-10-19")),
	)
	use := app.NewGetOffer([]app.OfferSource{source}, now)

	detail, err := use.Get(
		t.Context(),
		app.OfferQuery{Bank: bog, ID: "16902", Language: kernel.LanguageEnglish},
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if detail.Terms != "Mazda 3 Sedan: from 65 400 GEL" || !detail.ReadAt.Equal(readAt()) ||
		detail.Bank.Code != bog || detail.Offer.ID != "16902" {
		t.Errorf("Get() = %+v", detail)
	}

	if calls := source.termsCalls.Load(); calls != 0 {
		t.Errorf("Terms calls = %d, want 0", calls)
	}

	for _, id := range []string{"123", "ended"} {
		_, err := use.Get(t.Context(), app.OfferQuery{Bank: bog, ID: id})
		if !errors.Is(err, app.ErrOfferNotFound) {
			t.Errorf("Get(%s) error = %v, want ErrOfferNotFound", id, err)
		}
	}
}

// BOGOFFER-SC-015: An offer with no terms in the list reads them from the source.
func TestGetOfferReadsTheTerms(t *testing.T) {
	t.Parallel()

	source := newFake("tbc", offer(t, "7tD3", func(o *domain.Offer) { o.Bank = "tbc" }))
	use := app.NewGetOffer([]app.OfferSource{source}, now)

	detail, err := use.Get(t.Context(), app.OfferQuery{Bank: "tbc", ID: "7tD3"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if detail.Terms != "the terms of 7tD3" || source.termsCalls.Load() != 1 {
		t.Errorf("Get() terms = %q with %d calls", detail.Terms, source.termsCalls.Load())
	}

	if _, err := use.Get(
		t.Context(),
		app.OfferQuery{Bank: "abc", ID: "1"},
	); !errors.Is(
		err,
		app.ErrUnknownBank,
	) {
		t.Errorf("Get(abc) error = %v, want ErrUnknownBank", err)
	}
}
