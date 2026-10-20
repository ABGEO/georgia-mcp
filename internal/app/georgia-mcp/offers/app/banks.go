package app

import (
	"cmp"
	"slices"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
)

// banks holds the source of each bank of the server.
type banks struct {
	sources map[domain.BankCode]OfferSource
	list    []domain.Bank
}

func newBanks(sources []OfferSource) banks {
	set := banks{sources: make(map[domain.BankCode]OfferSource, len(sources))}

	for _, source := range sources {
		bank := source.Bank()
		set.sources[bank.Code] = source
		set.list = append(set.list, bank)
	}

	slices.SortFunc(set.list, func(a, b domain.Bank) int {
		return cmp.Compare(a.Code, b.Code)
	})

	return set
}

// resolve returns the requested banks sorted by code, or every bank for an empty list.
func (b banks) resolve(codes []domain.BankCode) ([]domain.Bank, error) {
	if len(codes) == 0 {
		return slices.Clone(b.list), nil
	}

	resolved := make([]domain.Bank, 0, len(codes))

	for _, code := range slices.Compact(slices.Sorted(slices.Values(codes))) {
		bank, err := b.find(code)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, bank)
	}

	return resolved, nil
}

// find returns the bank of a code, or an *UnknownBankError.
func (b banks) find(code domain.BankCode) (domain.Bank, error) {
	source, ok := b.sources[code]
	if !ok {
		known := make([]domain.BankCode, 0, len(b.list))
		for _, bank := range b.list {
			known = append(known, bank.Code)
		}

		return domain.Bank{}, &UnknownBankError{Code: code, Known: known}
	}

	return source.Bank(), nil
}

func bankNames(list []domain.Bank) string {
	names := make([]string, 0, len(list))
	for _, bank := range list {
		names = append(names, bank.Name)
	}

	return strings.Join(names, ", ")
}
