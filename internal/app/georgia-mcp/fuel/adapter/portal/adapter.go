// Package portal reads the fuel prices from the Portal client and converts its price
// list to the domain. It is the anti-corruption layer of the context for Portal.
package portal

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
	portalclient "github.com/abgeo/georgia-mcp/pkg/portal"
)

var errMalformed = errors.New("portal: malformed price list")

type product struct {
	grade    domain.Grade
	georgian string
}

// productOf returns the product of a slug of Portal. Portal gives the English name only,
// and its terms name the products in Georgian.
func productOf(slug string) (product, bool) {
	switch slug {
	case "super":
		return product{grade: domain.GradeSuper, georgian: "სუპერი"}, true
	case "premium":
		return product{grade: domain.GradePremium, georgian: "პრემიუმი"}, true
	case "euro-regular":
		return product{grade: domain.GradeRegular, georgian: "ევრო რეგულარი"}, true
	case "euro-diesel":
		return product{grade: domain.GradeDiesel, georgian: "ევრო დიზელი"}, true
	case "effect-diesel":
		return product{grade: domain.GradeDiesel, georgian: "ეფექტ დიზელი"}, true
	default:
		return product{}, false
	}
}

// Adapter implements app.PriceSource with the Portal client.
type Adapter struct {
	client *portalclient.Client
	now    func() time.Time
	logger *slog.Logger
	warned sync.Map
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each slug that the adapter cannot classify.
func NewAdapter(
	client *portalclient.Client,
	now func() time.Time,
	logger *slog.Logger,
) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Company returns Portal. Its page shows no date.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "portal", Name: "Portal", PublishesEffectiveDate: false}
}

// Prices reads every product with one call. It reads the standard price only: the
// online price of Portal is the price of a QR liter. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	entries, err := a.client.FuelPrices(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Portal prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	prices, err := a.toPrices(ctx, entries)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Portal prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	return app.PriceList{Prices: prices, ReadAt: a.now()}, nil
}

func (a *Adapter) toPrices(
	ctx context.Context,
	entries []portalclient.FuelPrice,
) ([]domain.ProductPrice, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: no product", errMalformed)
	}

	prices := make([]domain.ProductPrice, 0, len(entries))

	for _, entry := range entries {
		slug := entry.FuelType.Slug
		name := strings.TrimSpace(entry.Name)

		standard, err := decimal.NewFromString(entry.LatestPrice)
		if err != nil {
			return nil, fmt.Errorf("%w: %q price %q", errMalformed, slug, entry.LatestPrice)
		}

		known, ok := productOf(slug)
		if !ok {
			known = product{grade: domain.GradeOther}

			a.warnOnce(ctx, slug, name)
		}

		price, err := domain.NewProductPrice(
			slug,
			domain.ProductName{Georgian: known.georgian, English: name},
			known.grade,
			standard,
			decimal.NullDecimal{},
			kernel.Date{},
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errMalformed, err)
		}

		prices = append(prices, price)
	}

	return prices, nil
}

func (a *Adapter) warnOnce(ctx context.Context, slug, name string) {
	if _, seen := a.warned.LoadOrStore(slug, struct{}{}); !seen {
		a.logger.WarnContext(
			ctx,
			"Portal sells a product with no grade",
			slog.String("slug", slug),
			slog.String("name", name),
		)
	}
}
