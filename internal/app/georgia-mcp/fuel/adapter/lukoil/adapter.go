// Package lukoil reads the fuel prices from the Lukoil client and converts the newest row
// of its price history to the domain. It is the anti-corruption layer of the context for
// Lukoil.
package lukoil

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
	lukoilclient "github.com/abgeo/georgia-mcp/pkg/lukoil"
)

// minColumns is the number, one product, and the instant of the change.
const minColumns = 3

var errMalformed = errors.New("lukoil: malformed price table")

type product struct {
	code  string
	grade domain.Grade
}

// productOf returns the product of a name of the head. The page has no product code, so
// the name is the key.
func productOf(name string) (product, bool) {
	switch name {
	case "Super Ecto 100":
		return product{code: "super_ecto_100", grade: domain.GradeSuper100}, true
	case "Super Ecto":
		return product{code: "super_ecto", grade: domain.GradeSuper}, true
	case "Premium Avangard":
		return product{code: "premium_avangard", grade: domain.GradePremium}, true
	case "Euro Regular":
		return product{code: "euro_regular", grade: domain.GradeRegular}, true
	case "Euro Diesel":
		return product{code: "euro_diesel", grade: domain.GradeDiesel}, true
	default:
		return product{}, false
	}
}

// Adapter implements app.PriceSource with the Lukoil client.
type Adapter struct {
	client *lukoilclient.Client
	now    func() time.Time
	logger *slog.Logger
	warned sync.Map
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each product name that the adapter cannot classify.
func NewAdapter(
	client *lukoilclient.Client,
	now func() time.Time,
	logger *slog.Logger,
) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Company returns Lukoil. It publishes the instant of each change of its prices.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "lukoil", Name: "Lukoil", PublishesEffectiveDate: true}
}

// Prices reads the price history with one call, and returns the prices of the newest
// change. It leaves out each product at the price zero. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	table, err := a.client.PriceHistory(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Lukoil prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	prices, err := a.toPrices(ctx, table)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Lukoil prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	return app.PriceList{Prices: prices, ReadAt: a.now()}, nil
}

func (a *Adapter) toPrices(
	ctx context.Context,
	table lukoilclient.PriceTable,
) ([]domain.ProductPrice, error) {
	if len(table.Header) < minColumns {
		return nil, fmt.Errorf("%w: a head of %d cells", errMalformed, len(table.Header))
	}

	row, changed, err := newest(table.Rows)
	if err != nil {
		return nil, err
	}

	effective := kernel.Today(changed)
	names := table.Header[1 : len(table.Header)-1]
	prices := make([]domain.ProductPrice, 0, len(names))

	for i, name := range names {
		standard, err := decimal.NewFromString(row[i+1])
		if err != nil || standard.IsNegative() {
			return nil, fmt.Errorf("%w: %q price %q", errMalformed, name, row[i+1])
		}

		if standard.IsZero() {
			continue
		}

		known := a.productFor(ctx, name)

		price, err := domain.NewProductPrice(
			known.code,
			domain.ProductName{English: name},
			known.grade,
			standard,
			decimal.NullDecimal{},
			effective,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errMalformed, err)
		}

		prices = append(prices, price)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("%w: no product above zero", errMalformed)
	}

	return prices, nil
}

// newest returns the row with the latest instant of change, and that instant. Of two
// rows with one instant, it returns the later in the page. The order of the page is not
// trusted: it lists the oldest row first today.
func newest(rows [][]string) ([]string, time.Time, error) {
	if len(rows) == 0 {
		return nil, time.Time{}, fmt.Errorf("%w: no row", errMalformed)
	}

	var (
		latest  []string
		changed time.Time
	)

	for _, row := range rows {
		cell := row[len(row)-1]

		instant, err := time.ParseInLocation(time.DateTime, cell, kernel.TbilisiZone())
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("%w: the instant %q", errMalformed, cell)
		}

		if latest == nil || !instant.Before(changed) {
			latest, changed = row, instant
		}
	}

	return latest, changed, nil
}

// productFor returns the product of a name. An unknown name gets the grade other, a code
// from the name, and one warning.
func (a *Adapter) productFor(ctx context.Context, name string) product {
	if known, ok := productOf(name); ok {
		return known
	}

	if _, seen := a.warned.LoadOrStore(name, struct{}{}); !seen {
		a.logger.WarnContext(
			ctx,
			"Lukoil sells a product with no grade",
			slog.String("name", name),
		)
	}

	return product{
		code:  strings.ReplaceAll(strings.ToLower(name), " ", "_"),
		grade: domain.GradeOther,
	}
}
