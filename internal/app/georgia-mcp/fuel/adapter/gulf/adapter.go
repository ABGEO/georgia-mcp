// Package gulf reads the fuel prices from the Gulf client and converts the newest row of
// the tier Gulf to the domain. It is the anti-corruption layer of the context for Gulf.
package gulf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gulfclient "github.com/abgeo/georgia-mcp/pkg/gulf"
)

const (
	dateClass = "first"
	// tierClass marks a price of the tier Gulf. The page marks the tier Gulf+, which this
	// server does not read, with "gp_col".
	tierClass = "g_col"
)

var errMalformed = errors.New("gulf: malformed price table")

type product struct {
	code     string
	grade    domain.Grade
	georgian string
}

// productOf returns the product of an English name of the head. The page has no product
// code, so the name is the key. The Georgian names are the names of the page in Georgian.
func productOf(name string) (product, bool) {
	switch name {
	case "G-Force Super":
		return product{"g_force_super", domain.GradeSuper, "G-Force სუპერი"}, true
	case "G-Force Premium":
		return product{"g_force_premium", domain.GradePremium, "G-Force პრემიუმი"}, true
	case "G-Force Euro Regular":
		return product{"g_force_euro_regular", domain.GradeRegular, "G-Force ევრო რეგულარი"}, true
	case "G-Force Euro Diesel":
		return product{"g_force_euro_diesel", domain.GradeDiesel, "G-Force ევრო დიზელი"}, true
	case "Euro Diesel":
		return product{"euro_diesel", domain.GradeDiesel, "ევრო დიზელი"}, true
	default:
		return product{}, false
	}
}

// Adapter implements app.PriceSource with the Gulf client.
type Adapter struct {
	client *gulfclient.Client
	now    func() time.Time
	logger *slog.Logger
	warned sync.Map
}

var _ app.PriceSource = (*Adapter)(nil)

// NewAdapter returns an adapter that reads with the client. now gives the read instant.
// The logger receives one warning for each product name that the adapter cannot classify.
func NewAdapter(
	client *gulfclient.Client,
	now func() time.Time,
	logger *slog.Logger,
) *Adapter {
	return &Adapter{client: client, now: now, logger: logger}
}

// Company returns Gulf. It publishes the date of each change of its prices.
func (a *Adapter) Company() domain.Company {
	return domain.Company{Code: "gulf", Name: "Gulf", PublishesEffectiveDate: true}
}

// Prices reads the price archive with one call, and returns the prices of the tier Gulf
// of the newest change. It leaves out each product at the price zero. Every failure wraps
// kernel.ErrSourceUnavailable.
func (a *Adapter) Prices(ctx context.Context) (app.PriceList, error) {
	table, err := a.client.PriceArchive(ctx)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Gulf prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	prices, err := a.toPrices(ctx, table)
	if err != nil {
		return app.PriceList{}, fmt.Errorf(
			"reading the Gulf prices: %w: %w", kernel.ErrSourceUnavailable, err,
		)
	}

	return app.PriceList{Prices: prices, ReadAt: a.now()}, nil
}

func (a *Adapter) toPrices(
	ctx context.Context,
	table gulfclient.PriceTable,
) ([]domain.ProductPrice, error) {
	dateColumn := slices.IndexFunc(table.Columns, func(c gulfclient.Column) bool {
		return hasClass(c, dateClass)
	})
	if dateColumn < 0 {
		return nil, fmt.Errorf("%w: no date column", errMalformed)
	}

	tier := tierColumns(table.Columns)
	if len(tier) == 0 {
		return nil, fmt.Errorf("%w: no column of the tier Gulf", errMalformed)
	}

	row, effective, err := newest(table.Rows, dateColumn)
	if err != nil {
		return nil, err
	}

	prices := make([]domain.ProductPrice, 0, len(tier))

	for _, i := range tier {
		name := table.Columns[i].Name

		standard, err := decimal.NewFromString(row[i])
		if err != nil || standard.IsNegative() {
			return nil, fmt.Errorf("%w: %q price %q", errMalformed, name, row[i])
		}

		if standard.IsZero() {
			continue
		}

		known := a.productFor(ctx, name)

		price, err := domain.NewProductPrice(
			known.code,
			domain.ProductName{English: name, Georgian: known.georgian},
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

func hasClass(column gulfclient.Column, class string) bool {
	return slices.Contains(strings.Fields(column.Class), class)
}

// tierColumns returns the index of each column of the tier Gulf.
func tierColumns(columns []gulfclient.Column) []int {
	var tier []int

	for i, column := range columns {
		if hasClass(column, tierClass) {
			tier = append(tier, i)
		}
	}

	return tier
}

// newest returns the row with the latest date, and that date. Of two rows with one date,
// it returns the earlier in the page. The order of the page is not trusted: it lists the
// newest row first today.
func newest(rows [][]string, dateColumn int) ([]string, kernel.Date, error) {
	if len(rows) == 0 {
		return nil, kernel.Date{}, fmt.Errorf("%w: no row", errMalformed)
	}

	var (
		latest  []string
		changed kernel.Date
	)

	for _, row := range rows {
		date, err := kernel.ParseDate(row[dateColumn])
		if err != nil {
			return nil, kernel.Date{}, fmt.Errorf("%w: %w", errMalformed, err)
		}

		if latest == nil || date.After(changed) {
			latest, changed = row, date
		}
	}

	return latest, changed, nil
}

// productFor returns the product of a name. An unknown name gets the grade other, no
// Georgian name, a code from the name, and one warning.
func (a *Adapter) productFor(ctx context.Context, name string) product {
	if known, ok := productOf(name); ok {
		return known
	}

	if _, seen := a.warned.LoadOrStore(name, struct{}{}); !seen {
		a.logger.WarnContext(
			ctx,
			"Gulf sells a product with no grade",
			slog.String("name", name),
		)
	}

	return product{code: codeOf(name), grade: domain.GradeOther}
}

// codeOf returns the name in lower case, with "_" for each run of characters that are
// not a letter or a digit: "G-Force Hydro" gives "g_force_hydro".
func codeOf(name string) string {
	words := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	return strings.Join(words, "_")
}
