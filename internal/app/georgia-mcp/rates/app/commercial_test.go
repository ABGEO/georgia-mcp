package app_test

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

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	euro   domain.CurrencyCode = "EUR"
	noCode domain.CurrencyCode = "XXX"
)

func tbcBank() domain.Bank { return domain.Bank{Code: "tbc", Name: "TBC Bank"} }

func bogBank() domain.Bank { return domain.Bank{Code: "bog", Name: "Bank of Georgia"} }

// fakeBank quotes its codes at one price, and records each request.
type fakeBank struct {
	bank   domain.Bank
	quotes []domain.CurrencyCode
	delay  time.Duration
	fail   bool

	mu    sync.Mutex
	asked [][]domain.CurrencyCode
}

func (f *fakeBank) Bank() domain.Bank { return f.bank }

func (f *fakeBank) CommercialRates(
	_ context.Context,
	codes []domain.CurrencyCode,
) (app.CommercialRatesResult, error) {
	f.mu.Lock()
	f.asked = append(f.asked, slices.Clone(codes))
	f.mu.Unlock()

	time.Sleep(f.delay)

	if f.fail {
		return app.CommercialRatesResult{}, fmt.Errorf(
			"calling %s: %w",
			f.bank.Name,
			kernel.ErrSourceUnavailable,
		)
	}

	var result app.CommercialRatesResult

	for _, code := range codes {
		if !slices.Contains(f.quotes, code) {
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

func (f *fakeBank) requests() [][]domain.CurrencyCode {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.asked)
}

func newUseCase(banks ...*fakeBank) *app.GetCommercialRates {
	sources := make([]app.CommercialRateSource, 0, len(banks))
	for _, bank := range banks {
		sources = append(sources, bank)
	}

	return app.NewGetCommercialRates(sources)
}

func codesOf(answer app.BankRates) []domain.CurrencyCode {
	codes := make([]domain.CurrencyCode, 0, len(answer.Rates))
	for _, rate := range answer.Rates {
		codes = append(codes, rate.Currency)
	}

	return codes
}

func bankCodes(banks []app.BankRates) []domain.BankCode {
	codes := make([]domain.BankCode, 0, len(banks))
	for _, bank := range banks {
		codes = append(codes, bank.Bank.Code)
	}

	return codes
}

// TBCRATE-SC-007: The answer holds the requested currency of the requested bank only.
func TestGetCommercialRatesOfOneCurrency(t *testing.T) {
	t.Parallel()

	bank := &fakeBank{bank: tbcBank(), quotes: []domain.CurrencyCode{usd, euro}}

	got, err := newUseCase(bank).Get(t.Context(), app.CommercialRatesQuery{
		Currencies: []domain.CurrencyCode{usd},
		Banks:      []domain.BankCode{tbcBank().Code},
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(got.Banks) != 1 {
		t.Fatalf("Get() = %d banks, want 1", len(got.Banks))
	}

	if diff := cmp.Diff([]domain.CurrencyCode{usd}, codesOf(got.Banks[0])); diff != "" {
		t.Errorf("codes mismatch (-want +got):\n%s", diff)
	}

	if len(got.Banks[0].Missing) != 0 {
		t.Errorf("Missing = %v, want none", got.Banks[0].Missing)
	}
}

// TBCRATE-SC-008: A request with no bank asks every bank, sorted by code.
func TestGetCommercialRatesOfEveryBank(t *testing.T) {
	t.Parallel()

	quotes := []domain.CurrencyCode{usd}
	useCase := newUseCase(
		&fakeBank{bank: tbcBank(), quotes: quotes},
		&fakeBank{bank: bogBank(), quotes: quotes},
	)

	got, err := useCase.Get(t.Context(), app.CommercialRatesQuery{Currencies: quotes})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if diff := cmp.Diff(
		[]domain.BankCode{bogBank().Code, tbcBank().Code},
		bankCodes(got.Banks),
	); diff != "" {
		t.Errorf("banks mismatch (-want +got):\n%s", diff)
	}
}

// TBCRATE-SC-009: An unknown bank code is an UnknownBankError with the known codes, and
// no bank is asked.
func TestGetCommercialRatesRefusesAnUnknownBank(t *testing.T) {
	t.Parallel()

	bank := &fakeBank{bank: tbcBank(), quotes: []domain.CurrencyCode{usd}}

	_, err := newUseCase(bank).Get(t.Context(), app.CommercialRatesQuery{
		Currencies: []domain.CurrencyCode{usd},
		Banks:      []domain.BankCode{bogBank().Code},
	})

	want := &app.UnknownBankError{Code: bogBank().Code, Known: []domain.BankCode{tbcBank().Code}}

	got, ok := errors.AsType[*app.UnknownBankError](err)
	if !ok {
		t.Fatalf("Get() error = %v, want an UnknownBankError", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("UnknownBankError mismatch (-want +got):\n%s", diff)
	}

	if asked := bank.requests(); len(asked) != 0 {
		t.Errorf("the bank was asked %v, want no request", asked)
	}
}

// TBCRATE-SC-010: A code that the bank does not quote is in Missing, beside the other rates.
func TestGetCommercialRatesListsAMissingCode(t *testing.T) {
	t.Parallel()

	bank := &fakeBank{bank: tbcBank(), quotes: []domain.CurrencyCode{usd}}

	got, err := newUseCase(bank).Get(t.Context(), app.CommercialRatesQuery{
		Currencies: []domain.CurrencyCode{usd, noCode},
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if diff := cmp.Diff([]domain.CurrencyCode{usd}, codesOf(got.Banks[0])); diff != "" {
		t.Errorf("codes mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]domain.CurrencyCode{noCode}, got.Banks[0].Missing); diff != "" {
		t.Errorf("Missing mismatch (-want +got):\n%s", diff)
	}
}

// TBCRATE-SC-011: A bank that fails joins Unavailable. When every requested bank fails,
// Get fails.
func TestGetCommercialRatesWhenABankFails(t *testing.T) {
	t.Parallel()

	quotes := []domain.CurrencyCode{usd}
	query := app.CommercialRatesQuery{Currencies: quotes}

	t.Run("one of two fails", func(t *testing.T) {
		t.Parallel()

		useCase := newUseCase(
			&fakeBank{bank: tbcBank(), quotes: quotes, fail: true},
			&fakeBank{bank: bogBank(), quotes: quotes},
		)

		got, err := useCase.Get(t.Context(), query)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if diff := cmp.Diff([]domain.BankCode{bogBank().Code}, bankCodes(got.Banks)); diff != "" {
			t.Errorf("banks mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff([]domain.Bank{tbcBank()}, got.Unavailable); diff != "" {
			t.Errorf("Unavailable mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("the only bank fails", func(t *testing.T) {
		t.Parallel()

		useCase := newUseCase(&fakeBank{bank: tbcBank(), quotes: quotes, fail: true})

		_, err := useCase.Get(t.Context(), query)
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("Get() error = %v, want ErrSourceUnavailable", err)
		}
	})
}

// TBCRATE-SC-011: A slow bank does not hold the others.
func TestGetCommercialRatesAsksTheBanksAtOnce(t *testing.T) {
	t.Parallel()

	quotes := []domain.CurrencyCode{usd}

	synctest.Test(t, func(t *testing.T) {
		useCase := newUseCase(
			&fakeBank{bank: tbcBank(), quotes: quotes, delay: time.Second},
			&fakeBank{bank: bogBank(), quotes: quotes},
		)
		start := time.Now()

		got, err := useCase.Get(t.Context(), app.CommercialRatesQuery{Currencies: quotes})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if len(got.Banks) != 2 {
			t.Errorf("Get() = %d banks, want 2", len(got.Banks))
		}

		if elapsed := time.Since(start); elapsed >= 1500*time.Millisecond {
			t.Errorf("Get() took %v, want less than 1.5s", elapsed)
		}
	})
}

// TBCRATE-SC-022: A request needs 1 to 10 distinct codes. A code named twice counts once.
func TestGetCommercialRatesChecksTheCountOfCodes(t *testing.T) {
	t.Parallel()

	ten := tenCodes()

	tests := []struct {
		name      string
		codes     []domain.CurrencyCode
		wantCount int
		wantAsked [][]domain.CurrencyCode
	}{
		{name: "none", codes: nil, wantCount: 0},
		{name: "eleven", codes: append(ten, "JPY"), wantCount: 11},
		{name: "ten", codes: ten, wantAsked: [][]domain.CurrencyCode{ten}},
		{
			name:      "twice",
			codes:     []domain.CurrencyCode{usd, usd},
			wantAsked: [][]domain.CurrencyCode{{usd}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bank := &fakeBank{bank: tbcBank(), quotes: ten}
			query := app.CommercialRatesQuery{Currencies: tt.codes}

			_, err := newUseCase(bank).Get(t.Context(), query)
			if tt.wantAsked == nil {
				wantCountError(t, err, tt.wantCount)

				return
			}

			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if diff := cmp.Diff(tt.wantAsked, bank.requests()); diff != "" {
				t.Errorf("requests mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// tenCodes returns ten distinct codes, sorted.
func tenCodes() []domain.CurrencyCode {
	return []domain.CurrencyCode{"AED", "AUD", "CAD", "CHF", "CNY", "CZK", "DKK", euro, "GBP", usd}
}

func wantCountError(t *testing.T, err error, count int) {
	t.Helper()

	countErr, ok := errors.AsType[*app.CurrencyCountError](err)
	if !ok || countErr.Count != count {
		t.Fatalf("Get() error = %v, want a CurrencyCountError with %d", err, count)
	}
}
