package cache_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// countingBank answers an empty catalog read at the current instant, and counts the calls.
type countingBank struct {
	offerCalls atomic.Int32
	termsCalls atomic.Int32
	failing    atomic.Bool
	release    chan struct{}
}

func (b *countingBank) Bank() domain.Bank {
	return domain.Bank{Code: "bog", Name: "Bank of Georgia"}
}

func (b *countingBank) Cards() []domain.Card {
	return []domain.Card{{Code: "amex", Name: "Amex"}}
}

func (b *countingBank) Offers(_ context.Context, language kernel.Language) (app.Catalog, error) {
	b.offerCalls.Add(1)

	if b.release != nil {
		<-b.release
	}

	if b.failing.Load() {
		return app.Catalog{}, fmt.Errorf("calling BOG: %w", kernel.ErrSourceUnavailable)
	}

	return app.Catalog{Offers: []domain.Offer{{ID: string(language)}}, ReadAt: time.Now()}, nil
}

func (b *countingBank) Terms(
	_ context.Context,
	offer domain.Offer,
	language kernel.Language,
) (app.Terms, error) {
	b.termsCalls.Add(1)

	if b.failing.Load() {
		return app.Terms{}, fmt.Errorf("calling BOG: %w", kernel.ErrSourceUnavailable)
	}

	return app.Terms{Text: offer.ID + "/" + string(language), ReadAt: time.Now()}, nil
}

func newCache(t *testing.T, source app.OfferSource) *cache.OfferCache {
	t.Helper()

	offers, err := cache.New(source)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = offers.Close() })

	return offers
}

// BOGOFFER-SC-018: A catalog is fresh for 1 hour. While the bank fails, the cache
// returns the stale catalog for 24 more hours.
func TestOfferCacheKeepsACatalog(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		bank := &countingBank{}
		offers := newCache(t, bank)

		if _, err := offers.Offers(t.Context(), kernel.LanguageEnglish); err != nil {
			t.Fatalf("Offers() at 0 error = %v", err)
		}

		time.Sleep(30 * time.Minute)

		fresh, err := offers.Offers(t.Context(), kernel.LanguageEnglish)
		if err != nil || fresh.Stale || bank.offerCalls.Load() != 1 {
			t.Fatalf(
				"at 30m: %+v, %v, %d calls; want fresh with 1 call",
				fresh,
				err,
				bank.offerCalls.Load(),
			)
		}

		bank.failing.Store(true)
		time.Sleep(time.Hour)

		stale, err := offers.Offers(t.Context(), kernel.LanguageEnglish)
		if err != nil || !stale.Stale || !stale.ReadAt.Equal(start) || bank.offerCalls.Load() != 2 {
			t.Fatalf("at 1h30m: %+v, %v, %d calls; want stale of 0 with 2 calls",
				stale, err, bank.offerCalls.Load())
		}

		time.Sleep(time.Until(start.Add(25*time.Hour + 30*time.Minute)))

		if _, err := offers.Offers(
			t.Context(),
			kernel.LanguageEnglish,
		); !errors.Is(
			err,
			kernel.ErrSourceUnavailable,
		) {
			t.Errorf("at 25h30m error = %v, want ErrSourceUnavailable", err)
		}
	})
}

// BOGOFFER-SC-019: Asks that arrive at once share one call, and each language has its
// own entry.
func TestOfferCacheSharesOneCall(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		bank := &countingBank{release: make(chan struct{})}
		offers := newCache(t, bank)

		var wg sync.WaitGroup

		for range 20 {
			wg.Go(func() { _, _ = offers.Offers(t.Context(), kernel.LanguageEnglish) })
		}

		synctest.Wait()
		close(bank.release)
		wg.Wait()

		if calls := bank.offerCalls.Load(); calls != 1 {
			t.Errorf("calls after 20 asks in en = %d, want 1", calls)
		}

		catalog, err := offers.Offers(t.Context(), kernel.LanguageGeorgian)
		if err != nil || catalog.Offers[0].ID != "ka" || bank.offerCalls.Load() != 2 {
			t.Errorf("ka = %+v, %v, %d calls; want the ka catalog with 2 calls",
				catalog, err, bank.offerCalls.Load())
		}
	})
}

// BOGOFFER-SC-018: The terms of an offer are fresh for 1 hour, by id and language.
func TestOfferCacheKeepsTheTerms(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		bank := &countingBank{}
		offers := newCache(t, bank)
		offer := domain.Offer{ID: "7tD3"}

		for range 3 {
			terms, err := offers.Terms(t.Context(), offer, kernel.LanguageEnglish)
			if err != nil || terms.Text != "7tD3/en" {
				t.Fatalf("Terms() = %+v, %v", terms, err)
			}
		}

		if _, err := offers.Terms(t.Context(), offer, kernel.LanguageGeorgian); err != nil {
			t.Fatalf("Terms(ka) error = %v", err)
		}

		bank.failing.Store(true)
		time.Sleep(2 * time.Hour)

		terms, err := offers.Terms(t.Context(), offer, kernel.LanguageEnglish)
		if err != nil || !terms.Stale || bank.termsCalls.Load() != 3 {
			t.Errorf(
				"at 2h: %+v, %v, %d calls; want stale terms with 3 calls",
				terms,
				err,
				bank.termsCalls.Load(),
			)
		}

		if bank := offers.Bank(); bank.Code != "bog" || len(offers.Cards()) != 1 {
			t.Errorf("Bank() = %+v, Cards() = %v", bank, offers.Cards())
		}
	})
}
