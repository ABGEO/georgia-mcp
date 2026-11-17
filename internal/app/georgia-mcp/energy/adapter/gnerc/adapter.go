// Package gnerc reads the balance and the prices from the GNERC client and converts them
// to the domain. It is the anti-corruption layer of the context for GNERC.
package gnerc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gnercclient "github.com/abgeo/georgia-mcp/pkg/gnerc"
)

// The fixed part of the name of each workbook. The rest of a name holds the range of
// years, and changes each year.
const (
	balanceFile     = "ფაქტიური ბალანსი"
	balancingFile   = "საბალანსო ელექტროენერგიის ფასები"
	capacityFeeFile = "გარანტირებული სიმძლავრის საფასური"
	deregulatedFile = "დერეგულირებული სადგურების ფასები"
)

var (
	errNoLink   = errors.New("workbook: no link")
	errTwoLinks = errors.New("workbook: more than one link")
	errBadValue = errors.New("cell: not a number")
)

// Adapter implements app.BalanceSource and app.PriceSource over the GNERC client.
type Adapter struct {
	client *gnercclient.Client
	now    func() time.Time
}

var (
	_ app.BalanceSource = (*Adapter)(nil)
	_ app.PriceSource   = (*Adapter)(nil)
)

// NewAdapter returns an adapter over client. now gives the instant of each read.
func NewAdapter(client *gnercclient.Client, now func() time.Time) *Adapter {
	return &Adapter{client: client, now: now}
}

// Balance reads the links of the home page, then the workbook of the balance.
func (a *Adapter) Balance(ctx context.Context) (app.BalanceBook, error) {
	names, err := a.names(ctx, balanceFile)
	if err != nil {
		return app.BalanceBook{}, unavailable(err)
	}

	book, err := a.client.Workbook(ctx, names[0])
	if err != nil {
		return app.BalanceBook{}, unavailable(err)
	}

	years, err := balanceOf(book)
	if err != nil {
		return app.BalanceBook{}, unavailable(err)
	}

	return app.BalanceBook{
		Years:    years,
		Modified: modifiedOf(book),
		ReadAt:   a.now().In(kernel.TbilisiZone()),
	}, nil
}

// Prices reads the links of the home page, then the three workbooks of the prices, one
// at a time.
func (a *Adapter) Prices(ctx context.Context) (app.PriceBook, error) {
	names, err := a.names(ctx, balancingFile, capacityFeeFile, deregulatedFile)
	if err != nil {
		return app.PriceBook{}, unavailable(err)
	}

	var (
		series   [priceSeries]map[domain.Month]decimal.Decimal
		modified time.Time
	)

	for i, name := range names {
		book, err := a.client.Workbook(ctx, name)
		if err != nil {
			return app.PriceBook{}, unavailable(err)
		}

		series[i], err = pricesOf(book)
		if err != nil {
			return app.PriceBook{}, unavailable(fmt.Errorf("reading %q: %w", name, err))
		}

		if changed := modifiedOf(book); changed.After(modified) {
			modified = changed
		}
	}

	return app.PriceBook{
		Months:   joinPrices(series),
		Modified: modified,
		ReadAt:   a.now().In(kernel.TbilisiZone()),
	}, nil
}

// names returns the file name of the workbook of each fixed part, from the links of the
// home page.
func (a *Adapter) names(ctx context.Context, parts ...string) ([]string, error) {
	links, err := a.client.Links(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the links: %w", err)
	}

	names := make([]string, 0, len(parts))

	for _, part := range parts {
		var found []string

		for _, link := range links {
			address, err := url.Parse(link.Href)
			if err != nil {
				continue
			}

			name := path.Base(address.Path)
			if strings.Contains(collapse(name), part) && !slices.Contains(found, name) {
				found = append(found, name)
			}
		}

		switch len(found) {
		case 0:
			return nil, fmt.Errorf("%w: %q", errNoLink, part)
		case 1:
			names = append(names, found[0])
		default:
			return nil, fmt.Errorf("%w: %q", errTwoLinks, part)
		}
	}

	return names, nil
}

// collapse writes each run of spaces as one space, and trims the text.
func collapse(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// valueOf reads the text of a cell with each space removed, the non-breaking ones
// included, and rounds it.
func valueOf(text string) (decimal.Decimal, error) {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}

		return r
	}, text)

	number, err := decimal.NewFromString(clean)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("%w: %q", errBadValue, text)
	}

	return domain.Round(number), nil
}

func modifiedOf(book gnercclient.Workbook) time.Time {
	modified, err := http.ParseTime(book.LastModified)
	if err != nil {
		return time.Time{}
	}

	return modified.In(kernel.TbilisiZone())
}

func unavailable(err error) error {
	return fmt.Errorf("reading GNERC: %w: %w", kernel.ErrSourceUnavailable, err)
}
