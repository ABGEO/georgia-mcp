// Package wissol reads the fuel prices from the Wissol client and converts its answer
// to the domain. It is the anti-corruption layer of the context for Wissol.
package wissol

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	wissolclient "github.com/abgeo/georgia-mcp/pkg/wissol"
)

// dateLength is the length of the day at the start of the date of Wissol:
// "2026-10-08" of "2026-10-08T00:00:00".
const dateLength = len("2026-10-08")

var errMalformed = errors.New("wissol: malformed answer")

// gradeOfCode returns the grade of a product code of Wissol. The code stays when
// Wissol renames a product.
func gradeOfCode(code string) (domain.Grade, bool) {
	switch code {
	case "05":
		return domain.GradeRegular, true
	case "02":
		return domain.GradePremium, true
	case "01":
		return domain.GradeSuper, true
	case "07":
		return domain.GradeSuper100, true
	case "03", "06", "08":
		return domain.GradeDiesel, true
	case "09":
		return domain.GradeCNG, true
	default:
		return "", false
	}
}

// Adapter implements app.PriceSource with the Wissol client.
type Adapter struct {
	client *wissolclient.Client
	now    func() time.Time
	logger *slog.Logger
	warned sync.Map
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each product code that the adapter cannot classify.
func NewAdapter(client *wissolclient.Client, now func() time.Time, logger *slog.Logger) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Company returns Wissol.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "wissol", Name: "Wissol"}
}

// Prices reads every product with one call and leaves out each product that is not
// active. Every failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	products, err := a.client.FuelPrices(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Wissol prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	list := app.PriceList{Prices: make([]domain.ProductPrice, 0, len(products))}

	for _, product := range products {
		if !product.IsActive {
			continue
		}

		price, err := a.toPrice(ctx, product)
		if err != nil {
			return app.PriceList{}, fmt.Errorf(
				"reading the Wissol prices: %w: %w", kernel.ErrSourceUnavailable, err,
			)
		}

		list.Prices = append(list.Prices, price)
	}

	list.ReadAt = a.now()

	return list, nil
}

func (a *Adapter) toPrice(
	ctx context.Context,
	product wissolclient.FuelPrice,
) (domain.ProductPrice, error) {
	standard, err := decimal.NewFromString(product.Price.String())
	if err != nil {
		return domain.ProductPrice{}, fmt.Errorf(
			"%w: %q price %q", errMalformed, product.ProductCode, product.Price,
		)
	}

	selfService, err := toSelfService(product)
	if err != nil {
		return domain.ProductPrice{}, err
	}

	day := product.Date[:min(len(product.Date), dateLength)]

	effective, err := kernel.ParseDate(day)
	if err != nil {
		return domain.ProductPrice{}, fmt.Errorf(
			"%w: %q: %w",
			errMalformed,
			product.ProductCode,
			err,
		)
	}

	price, err := domain.NewProductPrice(
		product.ProductCode,
		domain.ProductName{Georgian: product.FuelType, English: product.FuelTypeEng},
		a.gradeOf(ctx, product),
		standard,
		selfService,
		effective,
	)
	if err != nil {
		return domain.ProductPrice{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return price, nil
}

// toSelfService reads an absent value, a null, and a zero as no self-service price.
func toSelfService(product wissolclient.FuelPrice) (decimal.NullDecimal, error) {
	if product.PriceSelfService == "" {
		return decimal.NullDecimal{}, nil
	}

	price, err := decimal.NewFromString(product.PriceSelfService.String())
	if err != nil || price.IsNegative() {
		return decimal.NullDecimal{}, fmt.Errorf(
			"%w: %q self-service price %q",
			errMalformed,
			product.ProductCode,
			product.PriceSelfService,
		)
	}

	if price.IsZero() {
		return decimal.NullDecimal{}, nil
	}

	return decimal.NewNullDecimal(price), nil
}

// gradeOf returns the grade of a product code, or other with one warning for each code.
func (a *Adapter) gradeOf(ctx context.Context, product wissolclient.FuelPrice) domain.Grade {
	if grade, ok := gradeOfCode(product.ProductCode); ok {
		return grade
	}

	if _, seen := a.warned.LoadOrStore(product.ProductCode, struct{}{}); !seen {
		a.logger.WarnContext(
			ctx,
			"Wissol sells a product with no grade",
			slog.String("code", product.ProductCode),
			slog.String("name", product.FuelTypeEng),
		)
	}

	return domain.GradeOther
}
