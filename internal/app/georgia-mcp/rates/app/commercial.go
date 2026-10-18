package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// CommercialRatesQuery asks for the commercial rates of some currencies at some banks.
type CommercialRatesQuery struct {
	// Currencies holds 1 to MaxCommercialCurrencies codes. A code named twice counts once.
	Currencies []domain.CurrencyCode
	// Banks is empty to ask every bank.
	Banks []domain.BankCode
}

// BankRates is the answer of one bank.
type BankRates struct {
	CommercialRatesResult

	Bank domain.Bank
}

// CommercialRates holds the answer of each bank that answered, sorted by bank code,
// and the banks that failed with no rates held, sorted by bank code.
type CommercialRates struct {
	Banks       []BankRates
	Unavailable []domain.Bank
}

// GetCommercialRates returns the commercial rates of the banks of the server.
type GetCommercialRates struct {
	sources map[domain.BankCode]CommercialRateSource
	banks   []domain.Bank
}

// NewGetCommercialRates returns the use case over one source for each bank.
func NewGetCommercialRates(sources []CommercialRateSource) *GetCommercialRates {
	use := &GetCommercialRates{
		sources: make(map[domain.BankCode]CommercialRateSource, len(sources)),
	}

	for _, source := range sources {
		bank := source.Bank()
		use.sources[bank.Code] = source
		use.banks = append(use.banks, bank)
	}

	slices.SortFunc(use.banks, compareBanks)

	return use
}

// Banks returns the banks of the server, sorted by code.
func (u *GetCommercialRates) Banks() []domain.Bank {
	return slices.Clone(u.banks)
}

// Get asks each requested bank at once and waits for all. A bank that fails joins
// Unavailable. When every requested bank fails, Get returns an error that wraps
// kernel.ErrSourceUnavailable.
func (u *GetCommercialRates) Get(
	ctx context.Context,
	query CommercialRatesQuery,
) (CommercialRates, error) {
	codes := slices.Compact(slices.Sorted(slices.Values(query.Currencies)))
	if len(codes) == 0 || len(codes) > MaxCommercialCurrencies {
		return CommercialRates{}, &CurrencyCountError{Count: len(codes)}
	}

	banks, err := u.resolve(query.Banks)
	if err != nil {
		return CommercialRates{}, err
	}

	results := make([]CommercialRatesResult, len(banks))
	errs := make([]error, len(banks))

	var wg sync.WaitGroup

	for i, bank := range banks {
		wg.Go(func() {
			results[i], errs[i] = u.sources[bank.Code].CommercialRates(ctx, codes)
		})
	}

	wg.Wait()

	var answer CommercialRates

	for i, bank := range banks {
		if errs[i] != nil {
			answer.Unavailable = append(answer.Unavailable, bank)

			continue
		}

		answer.Banks = append(
			answer.Banks,
			BankRates{CommercialRatesResult: results[i], Bank: bank},
		)
	}

	if len(answer.Banks) == 0 {
		return CommercialRates{}, fmt.Errorf(
			"%w: %s: %w", kernel.ErrSourceUnavailable, bankNames(banks), errors.Join(errs...),
		)
	}

	return answer, nil
}

// resolve returns the requested banks sorted by code, or every bank for an empty list.
func (u *GetCommercialRates) resolve(codes []domain.BankCode) ([]domain.Bank, error) {
	if len(codes) == 0 {
		return u.Banks(), nil
	}

	banks := make([]domain.Bank, 0, len(codes))

	for _, code := range slices.Compact(slices.Sorted(slices.Values(codes))) {
		source, ok := u.sources[code]
		if !ok {
			known := make([]domain.BankCode, 0, len(u.banks))
			for _, bank := range u.banks {
				known = append(known, bank.Code)
			}

			return nil, &UnknownBankError{Code: code, Known: known}
		}

		banks = append(banks, source.Bank())
	}

	return banks, nil
}

func compareBanks(a, b domain.Bank) int {
	return cmp.Compare(a.Code, b.Code)
}

func bankNames(banks []domain.Bank) string {
	names := make([]string, 0, len(banks))
	for _, bank := range banks {
		names = append(names, bank.Name)
	}

	return strings.Join(names, ", ")
}
