package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const maxCells = 1000

// TableRequest asks the description of a table.
type TableRequest struct {
	Code     domain.TableCode
	Language kernel.Language
}

// DescribeTable gives the title and the dimensions of a table.
type DescribeTable struct {
	source TableSource
}

// NewDescribeTable returns the use case.
func NewDescribeTable(source TableSource) *DescribeTable {
	return &DescribeTable{source: source}
}

// Describe returns the table with every value of each dimension.
func (d *DescribeTable) Describe(
	ctx context.Context,
	request TableRequest,
) (TableDescription, error) {
	description, err := d.source.Table(ctx, request.Code, request.Language)
	if err != nil {
		return TableDescription{}, fmt.Errorf("describing the table: %w", err)
	}

	return description, nil
}

// CellQuery asks the cells of a table for a selection. A dimension that the selection
// omits takes every value.
type CellQuery struct {
	Code      domain.TableCode
	Selection []DimensionSelection
	Language  kernel.Language
}

// CellRow is one cell: the text of its value on each dimension, and its figure.
type CellRow struct {
	Values []string
	Figure domain.Figure
}

// Cells is the cells of a selection.
type Cells struct {
	// Table holds the selected values of each dimension only.
	Table domain.Table
	// Rows follow the order of the table, the last dimension the fastest.
	Rows   []CellRow
	ReadAt time.Time
	Stale  bool
}

// GetCells gives the cells of a table for a selection.
type GetCells struct {
	source TableSource
}

// NewGetCells returns the use case.
func NewGetCells(source TableSource) *GetCells {
	return &GetCells{source: source}
}

// Get checks the selection against the table, then reads its cells.
func (g *GetCells) Get(ctx context.Context, query CellQuery) (Cells, error) {
	description, err := g.source.Table(ctx, query.Code, query.Language)
	if err != nil {
		return Cells{}, fmt.Errorf("describing the table: %w", err)
	}

	dimensions, err := resolve(description.Table.Dimensions, query.Selection)
	if err != nil {
		return Cells{}, err
	}

	selection := make([]DimensionSelection, 0, len(dimensions))
	for _, dimension := range dimensions {
		selection = append(selection, DimensionSelection{
			Dimension: dimension.Code, Values: valueCodes(dimension),
		})
	}

	set, err := g.source.Cells(ctx, query.Code, selection)
	if err != nil {
		return Cells{}, fmt.Errorf("reading the cells: %w", err)
	}

	table := description.Table
	table.Dimensions = dimensions

	return Cells{
		Table:  table,
		Rows:   rows(dimensions, set.Figures),
		ReadAt: set.ReadAt,
		Stale:  set.Stale || description.Stale,
	}, nil
}

// resolve returns each dimension of the table with its selected values only.
func resolve(
	dimensions []domain.Dimension,
	selection []DimensionSelection,
) ([]domain.Dimension, error) {
	chosen := map[string]DimensionSelection{}

	for _, item := range selection {
		if !slices.ContainsFunc(
			dimensions,
			func(d domain.Dimension) bool { return d.Code == item.Dimension },
		) {
			problem := newSelectionError(ProblemUnknownDimension, item.Dimension)
			problem.Accepted = dimensionCodes(dimensions)

			return nil, problem
		}

		if _, twice := chosen[item.Dimension]; twice {
			return nil, newSelectionError(ProblemDuplicate, item.Dimension)
		}

		if (len(item.Values) > 0) == (item.Last != 0) || item.Last < 0 {
			return nil, newSelectionError(ProblemValuesOrLast, item.Dimension)
		}

		chosen[item.Dimension] = item
	}

	resolved := make([]domain.Dimension, 0, len(dimensions))
	cells := 1

	for _, dimension := range dimensions {
		values, err := selectValues(dimension, chosen[dimension.Code])
		if err != nil {
			return nil, err
		}

		dimension.Values = values
		resolved = append(resolved, dimension)
		cells *= len(values)
	}

	if cells > maxCells {
		problem := newSelectionError(ProblemTooManyCells, "")
		problem.Cells = cells

		return nil, problem
	}

	return resolved, nil
}

func selectValues(
	dimension domain.Dimension,
	item DimensionSelection,
) ([]domain.DimensionValue, error) {
	switch {
	case item.Last > 0:
		return dimension.Values[max(len(dimension.Values)-item.Last, 0):], nil
	case len(item.Values) == 0:
		return dimension.Values, nil
	}

	values := make([]domain.DimensionValue, 0, len(item.Values))

	for _, code := range item.Values {
		index := slices.IndexFunc(
			dimension.Values,
			func(v domain.DimensionValue) bool { return v.Code == code },
		)
		if index < 0 {
			problem := newSelectionError(ProblemUnknownValue, dimension.Code)
			problem.Value = code
			problem.Accepted = valueCodes(dimension)

			return nil, problem
		}

		values = append(values, dimension.Values[index])
	}

	return values, nil
}

// rows walks every combination of the values, the last dimension the fastest.
func rows(dimensions []domain.Dimension, figures map[string]domain.Figure) []CellRow {
	result := []CellRow{}
	index := make([]int, len(dimensions))

	for {
		codes := make([]string, len(dimensions))
		texts := make([]string, len(dimensions))

		for i, dimension := range dimensions {
			if len(dimension.Values) == 0 {
				return result
			}

			codes[i] = dimension.Values[index[i]].Code
			texts[i] = dimension.Values[index[i]].Text
		}

		result = append(result, CellRow{Values: texts, Figure: figures[CellKey(codes)]})

		position := len(dimensions) - 1
		for position >= 0 {
			index[position]++
			if index[position] < len(dimensions[position].Values) {
				break
			}

			index[position] = 0
			position--
		}

		if position < 0 {
			return result
		}
	}
}

func dimensionCodes(dimensions []domain.Dimension) []string {
	codes := make([]string, 0, len(dimensions))
	for _, dimension := range dimensions {
		codes = append(codes, dimension.Code)
	}

	return codes
}

func valueCodes(dimension domain.Dimension) []string {
	codes := make([]string, 0, len(dimension.Values))
	for _, value := range dimension.Values {
		codes = append(codes, value.Code)
	}

	return codes
}
