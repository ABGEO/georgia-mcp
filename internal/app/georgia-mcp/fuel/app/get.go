package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// PricesQuery asks for the prices of some grades at some fuel companies.
type PricesQuery struct {
	// Grades is empty to ask every grade. A grade named twice counts once.
	Grades []domain.Grade
	// Companies is empty to ask every company. A code named twice counts once.
	Companies []domain.CompanyCode
}

// CompanyPrices is the answer of one fuel company. Its PriceList holds the prices of
// the requested grades only, sorted by grade, then by product code.
type CompanyPrices struct {
	PriceList

	Company domain.Company
	// Missing holds the requested grades that the company does not sell, in the
	// order of domain.Grades.
	Missing []domain.Grade
}

// Prices holds the answer of each fuel company that answered, sorted by code, and
// the companies that failed with no prices held, sorted by code.
type Prices struct {
	Companies   []CompanyPrices
	Unavailable []domain.Company
}

// GetPrices returns the fuel prices of the companies of the server.
type GetPrices struct {
	sources   map[domain.CompanyCode]PriceSource
	companies []domain.Company
}

// NewGetPrices returns the use case over one source for each fuel company.
func NewGetPrices(sources []PriceSource) *GetPrices {
	use := &GetPrices{sources: make(map[domain.CompanyCode]PriceSource, len(sources))}

	for _, source := range sources {
		company := source.Company()
		use.sources[company.Code] = source
		use.companies = append(use.companies, company)
	}

	slices.SortFunc(use.companies, compareCompanies)

	return use
}

// Companies returns the fuel companies of the server, sorted by code.
func (u *GetPrices) Companies() []domain.Company {
	return slices.Clone(u.companies)
}

// Get asks each requested company at once and waits for all. A company that fails
// joins Unavailable. When every requested company fails, Get returns an error that
// wraps kernel.ErrSourceUnavailable.
func (u *GetPrices) Get(ctx context.Context, query PricesQuery) (Prices, error) {
	companies, err := u.resolve(query.Companies)
	if err != nil {
		return Prices{}, err
	}

	lists := make([]PriceList, len(companies))
	errs := make([]error, len(companies))

	var wg sync.WaitGroup

	for i, company := range companies {
		wg.Go(func() {
			lists[i], errs[i] = u.sources[company.Code].Prices(ctx)
		})
	}

	wg.Wait()

	grades := requestedGrades(query.Grades)

	var answer Prices

	for i, company := range companies {
		if errs[i] != nil {
			answer.Unavailable = append(answer.Unavailable, company)

			continue
		}

		answer.Companies = append(answer.Companies, keep(company, lists[i], grades))
	}

	if len(answer.Companies) == 0 {
		return Prices{}, fmt.Errorf(
			"%w: %s: %w",
			kernel.ErrSourceUnavailable,
			companyNames(companies),
			errors.Join(errs...),
		)
	}

	return answer, nil
}

// resolve returns the requested companies sorted by code, or every company for an
// empty list.
func (u *GetPrices) resolve(codes []domain.CompanyCode) ([]domain.Company, error) {
	if len(codes) == 0 {
		return u.Companies(), nil
	}

	companies := make([]domain.Company, 0, len(codes))

	for _, code := range slices.Compact(slices.Sorted(slices.Values(codes))) {
		source, ok := u.sources[code]
		if !ok {
			known := make([]domain.CompanyCode, 0, len(u.companies))
			for _, company := range u.companies {
				known = append(known, company.Code)
			}

			return nil, &UnknownCompanyError{Code: code, Known: known}
		}

		companies = append(companies, source.Company())
	}

	return companies, nil
}

// requestedGrades returns the grades of a query in the order of domain.Grades, or
// every grade for an empty list.
func requestedGrades(grades []domain.Grade) []domain.Grade {
	if len(grades) == 0 {
		return domain.Grades()
	}

	return slices.DeleteFunc(domain.Grades(), func(grade domain.Grade) bool {
		return !slices.Contains(grades, grade)
	})
}

// keep returns the prices of the grades, sorted, and the grades that the list lacks.
func keep(company domain.Company, list PriceList, grades []domain.Grade) CompanyPrices {
	answer := CompanyPrices{ReadAt: list.ReadAt, Stale: list.Stale, Company: company}

	for _, grade := range grades {
		found := false

		for _, price := range list.Prices {
			if price.Grade == grade {
				answer.Prices = append(answer.Prices, price)
				found = true
			}
		}

		if !found {
			answer.Missing = append(answer.Missing, grade)
		}
	}

	slices.SortStableFunc(answer.Prices, func(a, b domain.ProductPrice) int {
		return cmp.Or(
			cmp.Compare(slices.Index(grades, a.Grade), slices.Index(grades, b.Grade)),
			cmp.Compare(a.Code, b.Code),
		)
	})

	return answer
}

func compareCompanies(a, b domain.Company) int {
	return cmp.Compare(a.Code, b.Code)
}

func companyNames(companies []domain.Company) string {
	names := make([]string, 0, len(companies))
	for _, company := range companies {
		names = append(names, company.Name)
	}

	return strings.Join(names, ", ")
}
