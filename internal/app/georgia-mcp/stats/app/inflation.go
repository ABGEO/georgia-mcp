package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const monthsInYear = 12

// InflationQuery asks the inflation of a month. A nil Month asks the newest month. An empty
// Region is the country, and an empty Group is the total.
type InflationQuery struct {
	Month    *domain.Month
	Region   domain.Region
	Group    domain.PriceGroup
	Language kernel.Language
}

// InflationResult is the inflation of a month, and the tables that it reads.
type InflationResult struct {
	Inflation domain.Inflation
	Tables    []domain.CatalogEntry
	ReadAt    time.Time
	Stale     bool
}

// GetInflation gives the inflation of a month from the consumer price index.
type GetInflation struct {
	source PriceIndexSource
	now    func() time.Time
}

// NewGetInflation returns the use case. now gives the instant that sets the current year.
func NewGetInflation(source PriceIndexSource, now func() time.Time) *GetInflation {
	return &GetInflation{source: source, now: now}
}

// Get returns the inflation of the month, the region, and the group.
func (g *GetInflation) Get(ctx context.Context, query InflationQuery) (InflationResult, error) {
	region, group := query.Region, query.Group
	if region == "" {
		region = domain.RegionGeorgia
	}

	if group == "" {
		group = domain.GroupTotal
	}

	indices, month, err := g.indicesOf(ctx, query.Month)
	if err != nil {
		return InflationResult{}, err
	}

	key := IndexKey{Region: region, Group: group, Month: month.Number()}
	inflation := domain.Inflation{
		Month:   month,
		Region:  region,
		Group:   group,
		Annual:  indices.Annual[key].Minus100(),
		Monthly: indices.Monthly[key].Minus100(),
		Average: indices.Average[key].Minus100(),
	}

	if region == domain.RegionGeorgia && group == domain.GroupTotal {
		inflation.Core = indices.Core[month.Number()]
	}

	return InflationResult{
		Inflation: inflation, Tables: indices.Tables, ReadAt: indices.ReadAt, Stale: indices.Stale,
	}, nil
}

// indicesOf returns the indices of the year of the month, or of the newest month.
func (g *GetInflation) indicesOf(
	ctx context.Context,
	month *domain.Month,
) (PriceIndexYear, domain.Month, error) {
	if month == nil {
		return g.newest(ctx)
	}

	indices, err := g.read(ctx, month.Year())
	if err != nil {
		return PriceIndexYear{}, domain.Month{}, err
	}

	if published(indices, month.Number()) {
		return indices, *month, nil
	}

	_, newest, err := g.newest(ctx)
	if err != nil && !errors.Is(err, ErrMonthNotPublished) {
		return PriceIndexYear{}, domain.Month{}, err
	}

	return PriceIndexYear{}, domain.Month{}, &MonthError{Month: *month, Newest: newest}
}

// newest returns the indices of the newest month with an annual index for the country,
// in the current year of Tbilisi or in the year before.
func (g *GetInflation) newest(ctx context.Context) (PriceIndexYear, domain.Month, error) {
	year := kernel.Today(g.now()).Time().Year()

	for _, candidate := range []int{year, year - 1} {
		indices, err := g.read(ctx, candidate)
		if err != nil {
			return PriceIndexYear{}, domain.Month{}, err
		}

		for number := monthsInYear; number >= 1; number-- {
			if published(indices, number) {
				month, err := domain.NewMonth(candidate, number)
				if err != nil {
					return PriceIndexYear{}, domain.Month{}, fmt.Errorf(
						"building the month: %w",
						err,
					)
				}

				return indices, month, nil
			}
		}
	}

	return PriceIndexYear{}, domain.Month{}, &MonthError{}
}

func (g *GetInflation) read(ctx context.Context, year int) (PriceIndexYear, error) {
	indices, err := g.source.PriceIndices(ctx, year)
	if err != nil {
		return PriceIndexYear{}, fmt.Errorf("reading the price indices of %d: %w", year, err)
	}

	return indices, nil
}

func published(indices PriceIndexYear, month int) bool {
	key := IndexKey{Region: domain.RegionGeorgia, Group: domain.GroupTotal, Month: month}

	return indices.Annual[key].Published()
}
