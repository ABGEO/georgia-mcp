package gnerc

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	gnercclient "github.com/abgeo/georgia-mcp/pkg/gnerc"
)

const (
	priceSeries      = 3
	monthColumn      = "A"
	priceHeader      = "თვე"
	balancingIndex   = 0
	capacityFeeIndex = 1
	deregulatedIndex = 2
)

// pricesOf reads the first sheet of a price workbook: a header row of years, then a row
// for each month. An empty cell gives no price.
func pricesOf(book gnercclient.Workbook) (map[domain.Month]decimal.Decimal, error) {
	if len(book.Sheets) == 0 {
		return nil, errNoHeader
	}

	rows := book.Sheets[0].Rows
	header := slices.IndexFunc(rows, func(row gnercclient.Row) bool {
		return cellText(row, monthColumn) == priceHeader
	})

	if header < 0 {
		return nil, errNoHeader
	}

	years, err := yearsOf(rows[header])
	if err != nil {
		return nil, err
	}

	names := monthNames()
	prices := map[domain.Month]decimal.Decimal{}

	for _, row := range rows[header+1:] {
		calendar, found := names[cellText(row, monthColumn)]
		if !found {
			continue
		}

		for _, cell := range row.Cells {
			year, found := years[cell.Column]
			if !found || strings.TrimSpace(cell.Text) == "" {
				continue
			}

			value, err := valueOf(cell.Text)
			if err != nil {
				return nil, fmt.Errorf("row %d: %w", row.Number, err)
			}

			prices[domain.Month{Year: year, Month: calendar}] = value
		}
	}

	return prices, nil
}

// joinPrices returns each month from domain.FirstPriceYear with a price in a series, in
// order.
func joinPrices(series [priceSeries]map[domain.Month]decimal.Decimal) []domain.MonthPrices {
	published := map[domain.Month]bool{}

	for _, prices := range series {
		for month := range prices {
			if month.Year >= domain.FirstPriceYear {
				published[month] = true
			}
		}
	}

	months := slices.SortedFunc(maps.Keys(published), domain.Month.Compare)

	joined := make([]domain.MonthPrices, 0, len(months))

	for _, m := range months {
		joined = append(joined, domain.MonthPrices{
			Month:       m,
			Balancing:   priceAt(series[balancingIndex], m),
			CapacityFee: priceAt(series[capacityFeeIndex], m),
			Deregulated: priceAt(series[deregulatedIndex], m),
		})
	}

	return joined
}

func priceAt(prices map[domain.Month]decimal.Decimal, m domain.Month) *decimal.Decimal {
	price, found := prices[m]
	if !found {
		return nil
	}

	return &price
}

func cellText(row gnercclient.Row, column string) string {
	for _, cell := range row.Cells {
		if cell.Column == column {
			return collapse(cell.Text)
		}
	}

	return ""
}

// yearsOf returns the year of each column of the header row.
func yearsOf(header gnercclient.Row) (map[string]int, error) {
	years := map[string]int{}

	for _, cell := range header.Cells {
		if cell.Column == monthColumn {
			continue
		}

		year, err := strconv.Atoi(collapse(cell.Text))
		if err != nil {
			return nil, fmt.Errorf("%w: a year %q", errNoColumn, cell.Text)
		}

		years[cell.Column] = year
	}

	return years, nil
}
