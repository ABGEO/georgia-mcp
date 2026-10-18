package cache_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	usd domain.CurrencyCode = "USD"
	eur domain.CurrencyCode = "EUR"
	krw domain.CurrencyCode = "KRW"
)

// countingBank quotes USD and EUR, and records each request.
type countingBank struct {
	mu      sync.Mutex
	asked   [][]domain.CurrencyCode
	failing bool
	release chan struct{}
}

func (b *countingBank) Bank() domain.Bank { return domain.Bank{Code: "tbc", Name: "TBC Bank"} }

func (b *countingBank) CommercialRates(
	_ context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	b.mu.Lock()
	b.asked = append(b.asked, slices.Clone(codes))
	failing, release := b.failing, b.release
	b.mu.Unlock()

	if release != nil {
		<-release
	}

	if failing {
		return app.CommercialRatesResult{}, fmt.Errorf(
			"calling TBC: %w",
			kernel.ErrSourceUnavailable,
		)
	}

	result := app.CommercialRatesResult{ReadAt: time.Now()}

	for _, code := range codes {
		if code != usd && code != eur {
			result.Missing = append(result.Missing, code)

			continue
		}

		rate, _ := domain.NewCommercialRate(
			code, decimal.RequireFromString("2.566"), decimal.RequireFromString("2.628"),
		)
		result.Rates = append(result.Rates, rate)
	}

	return result, nil
}

func (b *countingBank) fail() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failing = true
}

func (b *countingBank) requests() [][]domain.CurrencyCode {
	b.mu.Lock()
	defer b.mu.Unlock()

	return slices.Clone(b.asked)
}

func newCommercial(t *testing.T, bank *countingBank) *cache.CommercialRateCache {
	t.Helper()

	rates, err := cache.NewCommercial(bank)
	if err != nil {
		t.Fatalf("NewCommercial() error = %v", err)
	}

	t.Cleanup(func() { _ = rates.Close() })

	return rates
}

func askCommercial(
	t *testing.T,
	rates *cache.CommercialRateCache,
	codes ...domain.CurrencyCode,
) app.CommercialRatesResult {
	t.Helper()

	result, err := rates.CommercialRates(t.Context(), codes)
	if err != nil {
		t.Fatalf("CommercialRates(%v) error = %v", codes, err)
	}

	return result
}

// TBCRATE-SC-015: During an outage, the cache returns the codes it holds inside 1 hour,
// marked as stale. A code it never read fails the bank.
func TestCommercialRatesDuringAnOutage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		askedAt time.Duration
		codes   []domain.CurrencyCode
		wantErr bool
	}{
		{name: "held at 30 minutes", askedAt: 30 * time.Minute, codes: []domain.CurrencyCode{usd}},
		{
			name:    "past 1 hour",
			askedAt: 61 * time.Minute,
			codes:   []domain.CurrencyCode{usd},
			wantErr: true,
		},
		{
			name:    "a code never read",
			askedAt: 30 * time.Minute,
			codes:   []domain.CurrencyCode{usd, eur},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				bank := &countingBank{}
				rates := newCommercial(t, bank)
				readAt := time.Now()

				askCommercial(t, rates, usd)
				time.Sleep(time.Minute)
				bank.fail()
				time.Sleep(tt.askedAt - time.Minute)

				got, err := rates.CommercialRates(t.Context(), tt.codes)
				if tt.wantErr {
					if !errors.Is(err, kernel.ErrSourceUnavailable) {
						t.Errorf("CommercialRates() error = %v, want ErrSourceUnavailable", err)
					}

					return
				}

				if err != nil || !got.Stale || !got.ReadAt.Equal(readAt) || len(got.Rates) != 1 {
					t.Errorf(
						"CommercialRates() = %+v, %v, want USD stale, read at %v",
						got,
						err,
						readAt,
					)
				}
			})
		})
	}
}

// TBCRATE-SC-016: A rate stays 5 minutes.
func TestCommercialRatesStayFiveMinutes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		bank := &countingBank{}
		rates := newCommercial(t, bank)

		for _, wait := range []time.Duration{0, 4 * time.Minute, 2 * time.Minute} {
			time.Sleep(wait)

			got := askCommercial(t, rates, usd)
			if age := time.Since(got.ReadAt); age >= 5*time.Minute {
				t.Errorf("the rate is %v old, want less than 5 minutes", age)
			}
		}

		if got := len(bank.requests()); got != 2 {
			t.Errorf("the bank counted %d calls, want 2", got)
		}
	})
}

// TBCRATE-SC-017: Concurrent requests for the same codes share one call.
func TestCommercialRatesShareOneCall(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		bank := &countingBank{release: make(chan struct{})}
		rates := newCommercial(t, bank)

		var wg sync.WaitGroup

		for range 10 {
			wg.Go(func() { askCommercial(t, rates, usd, eur) })
		}

		synctest.Wait()
		close(bank.release)
		wg.Wait()

		if got := len(bank.requests()); got != 1 {
			t.Errorf("the bank counted %d calls, want 1", got)
		}
	})
}

// TBCRATE-SC-023: The cache asks only for the codes it lacks, keeps a code that the
// bank does not quote, and gives the oldest read instant.
func TestCommercialRatesAskForTheLackingCodes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		bank := &countingBank{}
		rates := newCommercial(t, bank)
		readAt := time.Now()

		askCommercial(t, rates, usd, krw)
		time.Sleep(2 * time.Minute)

		got := askCommercial(t, rates, usd, eur, krw)

		want := [][]domain.CurrencyCode{{krw, usd}, {eur}}
		if diff := cmp.Diff(want, bank.requests()); diff != "" {
			t.Errorf("requests mismatch (-want +got):\n%s", diff)
		}

		if !got.ReadAt.Equal(readAt) {
			t.Errorf("ReadAt = %v, want %v", got.ReadAt, readAt)
		}

		if diff := cmp.Diff([]domain.CurrencyCode{krw}, got.Missing); diff != "" {
			t.Errorf("Missing mismatch (-want +got):\n%s", diff)
		}
	})
}
