// Package neogas reads the gas price from the Neogas client and converts it to the
// domain. It is the anti-corruption layer of the context for Neogas.
package neogas

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	neogasclient "github.com/abgeo/georgia-mcp/pkg/neogas"
)

var errMalformed = errors.New("neogas: malformed gas price")

// Adapter implements app.PriceSource with the Neogas client.
type Adapter struct {
	client *neogasclient.Client
	now    func() time.Time
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
func NewAdapter(client *neogasclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Company returns Neogas. Its page shows no date.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "neogas", Name: "Neogas", PublishesEffectiveDate: false}
}

// Prices reads the one product of Neogas, CNG. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	texts, err := a.client.GasPrices(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Neogas price: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	price, err := toPrice(texts)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Neogas price: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	return app.PriceList{Prices: []domain.ProductPrice{price}, ReadAt: a.now()}, nil
}

// toPrice needs one value or more, all equal as decimals. Each pane of the calculator
// repeats the price, so two values that differ give no single price.
func toPrice(texts []string) (domain.ProductPrice, error) {
	if len(texts) == 0 {
		return domain.ProductPrice{}, fmt.Errorf("%w: no gas price", errMalformed)
	}

	var standard decimal.Decimal

	for i, text := range texts {
		value, err := decimal.NewFromString(text)
		if err != nil {
			return domain.ProductPrice{}, fmt.Errorf("%w: %q", errMalformed, text)
		}

		if i == 0 {
			standard = value

			continue
		}

		if !value.Equal(standard) {
			return domain.ProductPrice{}, fmt.Errorf(
				"%w: %s and %s differ", errMalformed, standard, value,
			)
		}
	}

	price, err := domain.NewProductPrice(
		"cng",
		domain.ProductName{Georgian: "ბუნებრივი აირი", English: "CNG"},
		domain.GradeCNG,
		standard,
		decimal.NullDecimal{},
		kernel.Date{},
	)
	if err != nil {
		return domain.ProductPrice{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return price, nil
}
