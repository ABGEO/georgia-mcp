// Package rompetrol reads the fuel prices from the Rompetrol client and converts the
// rows of its price table to the domain. It is the anti-corruption layer of the context
// for Rompetrol.
package rompetrol

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	rompetrolclient "github.com/abgeo/georgia-mcp/pkg/rompetrol"
)

// selfServicePrefix starts the name of a row that holds the self-service price of the
// product row directly before it.
const selfServicePrefix = "- "

var errMalformed = errors.New("rompetrol: malformed price table")

type product struct {
	code    string
	grade   domain.Grade
	english string
}

// productOf returns the product of a Georgian name of Rompetrol. The page has no
// product code, so the name is the key.
func productOf(georgian string) (product, bool) {
	switch georgian {
	case "efix სუპერი":
		return product{code: "efix_super", grade: domain.GradeSuper, english: "efix Super"}, true
	case "efix ევრო პრემიუმი":
		return product{
			code: "efix_euro_premium", grade: domain.GradePremium, english: "efix Euro Premium",
		}, true
	case "efix ევრო რეგულარი":
		return product{
			code: "efix_euro_regular", grade: domain.GradeRegular, english: "efix Euro Regular",
		}, true
	case "efix ევრო დიზელი":
		return product{
			code: "efix_euro_diesel", grade: domain.GradeDiesel, english: "efix Euro Diesel",
		}, true
	case "ევრო დიზელი":
		return product{code: "euro_diesel", grade: domain.GradeDiesel, english: "Euro Diesel"}, true
	default:
		return product{}, false
	}
}

// Adapter implements app.PriceSource with the Rompetrol client.
type Adapter struct {
	client *rompetrolclient.Client
	now    func() time.Time
	logger *slog.Logger
	warned sync.Map
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each product name that the adapter cannot classify.
func NewAdapter(
	client *rompetrolclient.Client,
	now func() time.Time,
	logger *slog.Logger,
) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Company returns Rompetrol. It publishes no effective date.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "rompetrol", Name: "Rompetrol", PublishesEffectiveDate: false}
}

// Prices reads the price table with one call. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	rows, err := a.client.Prices(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Rompetrol prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	prices, err := a.toPrices(ctx, rows)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Rompetrol prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	return app.PriceList{Prices: prices, ReadAt: a.now()}, nil
}

// pending is a product row that waits for the self-service row that can follow it.
type pending struct {
	row         rompetrolclient.PriceRow
	selfService decimal.NullDecimal
	paired      bool
}

func (a *Adapter) toPrices(
	ctx context.Context,
	rows []rompetrolclient.PriceRow,
) ([]domain.ProductPrice, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: no row", errMalformed)
	}

	var products []*pending

	for _, row := range rows {
		if !strings.HasPrefix(row.Product, selfServicePrefix) {
			products = append(products, &pending{row: row})

			continue
		}

		if len(products) == 0 || products[len(products)-1].paired {
			return nil, fmt.Errorf("%w: %q follows no product", errMalformed, row.Product)
		}

		selfService, err := toSelfService(row)
		if err != nil {
			return nil, err
		}

		last := products[len(products)-1]
		last.selfService = selfService
		last.paired = true
	}

	prices := make([]domain.ProductPrice, 0, len(products))

	for _, item := range products {
		price, err := a.toPrice(ctx, item)
		if err != nil {
			return nil, err
		}

		prices = append(prices, price)
	}

	return prices, nil
}

func (a *Adapter) toPrice(ctx context.Context, item *pending) (domain.ProductPrice, error) {
	standard, err := decimal.NewFromString(item.row.Price)
	if err != nil {
		return domain.ProductPrice{}, fmt.Errorf(
			"%w: %q price %q", errMalformed, item.row.Product, item.row.Price,
		)
	}

	known := a.productOf(ctx, item.row.Product)

	price, err := domain.NewProductPrice(
		known.code,
		domain.ProductName{Georgian: item.row.Product, English: known.english},
		known.grade,
		standard,
		item.selfService,
		kernel.Date{},
	)
	if err != nil {
		return domain.ProductPrice{}, fmt.Errorf("%w: %w", errMalformed, err)
	}

	return price, nil
}

// toSelfService reads a zero as no self-service price.
func toSelfService(row rompetrolclient.PriceRow) (decimal.NullDecimal, error) {
	price, err := decimal.NewFromString(row.Price)
	if err != nil || price.IsNegative() {
		return decimal.NullDecimal{}, fmt.Errorf(
			"%w: %q self-service price %q", errMalformed, row.Product, row.Price,
		)
	}

	if price.IsZero() {
		return decimal.NullDecimal{}, nil
	}

	return decimal.NewNullDecimal(price), nil
}

// productOf returns the known product of a name, or a product of the grade other with
// the name as its code and one warning for each name.
func (a *Adapter) productOf(ctx context.Context, georgian string) product {
	if known, ok := productOf(georgian); ok {
		return known
	}

	if _, seen := a.warned.LoadOrStore(georgian, struct{}{}); !seen {
		a.logger.WarnContext(
			ctx,
			"Rompetrol sells a product with no grade",
			slog.String("name", georgian),
		)
	}

	return product{code: georgian, grade: domain.GradeOther}
}
