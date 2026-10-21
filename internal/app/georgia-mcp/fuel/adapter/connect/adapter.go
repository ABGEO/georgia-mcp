// Package connect reads the fuel prices from the Connect client and converts its answer
// to the domain. It is the anti-corruption layer of the context for Connect.
package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	connectclient "github.com/abgeo/georgia-mcp/pkg/connect"
)

var errMalformed = errors.New("connect: malformed answer")

type product struct {
	grade   domain.Grade
	english string
}

// productOf returns the product of a key of Connect. The key stays when Connect renames a
// product.
func productOf(key string) (product, bool) {
	switch key {
	case "super_ron98":
		return product{grade: domain.GradeSuper, english: "Super RON 98"}, true
	case "ron96":
		return product{grade: domain.GradePremium, english: "Premium RON 96"}, true
	case "regular_ron93":
		return product{grade: domain.GradeRegular, english: "Regular RON 93"}, true
	case "euro_diesel_10ppm":
		return product{grade: domain.GradeDiesel, english: "Euro Diesel 10PPM"}, true
	case "diesel_10ppm":
		return product{grade: domain.GradeDiesel, english: "Diesel 10PPM"}, true
	default:
		return product{}, false
	}
}

// Adapter implements app.PriceSource with the Connect client.
type Adapter struct {
	client *connectclient.Client
	now    func() time.Time
	logger *slog.Logger
	warned sync.Map
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each key that the adapter cannot classify.
func NewAdapter(
	client *connectclient.Client,
	now func() time.Time,
	logger *slog.Logger,
) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Company returns Connect. Its date is the day of a snapshot, so it publishes none.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "connect", Name: "Connect", PublishesEffectiveDate: false}
}

// Prices reads every product with one call. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	snapshot, err := a.client.Current(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Connect prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	prices, err := a.toPrices(ctx, snapshot)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Connect prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	return app.PriceList{Prices: prices, ReadAt: a.now()}, nil
}

func (a *Adapter) toPrices(
	ctx context.Context,
	snapshot connectclient.Snapshot,
) ([]domain.ProductPrice, error) {
	if len(snapshot.Prices) == 0 {
		return nil, fmt.Errorf("%w: no product", errMalformed)
	}

	prices := make([]domain.ProductPrice, 0, len(snapshot.Prices))

	for key, fuel := range snapshot.Prices {
		standard, err := shortestDecimal(fuel.Standard)
		if err != nil {
			return nil, fmt.Errorf("%w: %q standard: %w", errMalformed, key, err)
		}

		selfService, err := selfServiceOf(fuel.SelfService)
		if err != nil {
			return nil, fmt.Errorf("%w: %q self-service: %w", errMalformed, key, err)
		}

		known, ok := productOf(key)
		if !ok {
			known = product{grade: domain.GradeOther}

			a.warnOnce(ctx, key, fuel.Label)
		}

		price, err := domain.NewProductPrice(
			key,
			domain.ProductName{Georgian: strings.TrimSpace(fuel.Label), English: known.english},
			known.grade,
			standard,
			selfService,
			kernel.Date{},
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errMalformed, err)
		}

		prices = append(prices, price)
	}

	return prices, nil
}

// shortestDecimal reads a price that Connect sends as the full digits of a binary float.
// The shortest text that parses to the same float64 is the price that Connect entered.
func shortestDecimal(number json.Number) (decimal.Decimal, error) {
	float, err := strconv.ParseFloat(number.String(), 64)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("parsing %q: %w", number, err)
	}

	value, err := decimal.NewFromString(strconv.FormatFloat(float, 'f', -1, 64))
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("parsing %q: %w", number, err)
	}

	return value, nil
}

// selfServiceOf gives no price for an empty field or a zero, and fails on a negative one.
func selfServiceOf(number json.Number) (decimal.NullDecimal, error) {
	if number == "" {
		return decimal.NullDecimal{}, nil
	}

	value, err := shortestDecimal(number)
	if err != nil {
		return decimal.NullDecimal{}, err
	}

	switch {
	case value.IsNegative():
		return decimal.NullDecimal{}, fmt.Errorf("%w: %s", errMalformed, value)
	case value.IsZero():
		return decimal.NullDecimal{}, nil
	default:
		return decimal.NewNullDecimal(value), nil
	}
}

func (a *Adapter) warnOnce(ctx context.Context, key, label string) {
	if _, seen := a.warned.LoadOrStore(key, struct{}{}); !seen {
		a.logger.WarnContext(
			ctx,
			"Connect sells a product with no grade",
			slog.String("key", key),
			slog.String("label", strings.TrimSpace(label)),
		)
	}
}
