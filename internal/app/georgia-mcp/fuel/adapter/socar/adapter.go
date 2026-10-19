// Package socar reads the fuel prices from the SOCAR client and converts its answer to
// the domain. It is the anti-corruption layer of the context for SOCAR.
package socar

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
	socarclient "github.com/abgeo/georgia-mcp/pkg/socar"
)

const successStatus = "1"

var errMalformed = errors.New("socar: malformed answer")

// gradeOfCode returns the grade of a product code of SOCAR. The code stays when SOCAR
// renames a product.
func gradeOfCode(code string) (domain.Grade, bool) {
	switch code {
	case "SUPER":
		return domain.GradeSuper, true
	case "PREMIUM":
		return domain.GradePremium, true
	case "EURREG":
		return domain.GradeRegular, true
	case "DIESEL", "EURODSL":
		return domain.GradeDiesel, true
	case "LPG":
		return domain.GradeLPG, true
	case "CNG":
		return domain.GradeCNG, true
	default:
		return "", false
	}
}

// Adapter implements app.PriceSource with the SOCAR client.
type Adapter struct {
	client *socarclient.Client
	now    func() time.Time
	logger *slog.Logger
	warned sync.Map
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each product code that the adapter cannot classify.
func NewAdapter(client *socarclient.Client, now func() time.Time, logger *slog.Logger) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Company returns SOCAR. Its dates are not the dates of a change, so it publishes none.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "socar", Name: "SOCAR", PublishesEffectiveDate: false}
}

// Prices reads every product with one call and leaves out each product at the price
// zero. Every failure wraps kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	answer, err := a.client.CurrentPrices(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the SOCAR prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	prices, err := a.toPrices(ctx, answer)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the SOCAR prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	return app.PriceList{Prices: prices, ReadAt: a.now()}, nil
}

func (a *Adapter) toPrices(
	ctx context.Context,
	answer socarclient.PriceList,
) ([]domain.ProductPrice, error) {
	if answer.Status != successStatus {
		return nil, fmt.Errorf("%w: status %q", errMalformed, answer.Status)
	}

	prices := make([]domain.ProductPrice, 0, len(answer.Results))

	for _, product := range answer.Results {
		standard, err := decimal.NewFromString(product.FuelUnitPrice.String())
		if err != nil || standard.IsNegative() {
			return nil, fmt.Errorf(
				"%w: %q price %q", errMalformed, product.FuelCode, product.FuelUnitPrice,
			)
		}

		if standard.IsZero() {
			continue
		}

		price, err := domain.NewProductPrice(
			product.FuelCode,
			domain.ProductName{
				Georgian: strings.TrimSpace(product.FuelNameGeo),
				English:  strings.TrimSpace(product.FuelNameEng),
			},
			a.gradeOf(ctx, product),
			standard,
			decimal.NullDecimal{},
			kernel.Date{},
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

// gradeOf returns the grade of a product code, or other with one warning for each code.
func (a *Adapter) gradeOf(ctx context.Context, product socarclient.FuelPrice) domain.Grade {
	if grade, ok := gradeOfCode(product.FuelCode); ok {
		return grade
	}

	if _, seen := a.warned.LoadOrStore(product.FuelCode, struct{}{}); !seen {
		a.logger.WarnContext(
			ctx,
			"SOCAR sells a product with no grade",
			slog.String("code", product.FuelCode),
			slog.String("name", strings.TrimSpace(product.FuelNameEng)),
		)
	}

	return domain.GradeOther
}
