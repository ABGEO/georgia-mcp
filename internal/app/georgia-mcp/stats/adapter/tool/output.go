package tool

import (
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	source     = "Geostat"
	dateLayout = "2006-01-02"
)

func toTablesOutput(matches app.TableMatches) tablesOutput {
	tables := make([]tableOutput, 0, len(matches.Entries))
	for _, entry := range matches.Entries {
		tables = append(tables, toEntryOutput(entry))
	}

	return tablesOutput{
		Source: source,
		AsOf:   instant(matches.ReadAt),
		Stale:  matches.Stale,
		Total:  matches.Total,
		Tables: tables,
	}
}

func toEntryOutput(entry domain.CatalogEntry) tableOutput {
	return tableOutput{
		Code:    entry.Code.String(),
		Title:   entry.Title,
		Folder:  entry.Folder,
		Updated: entry.Updated.In(kernel.TbilisiZone()).Format(dateLayout),
	}
}

func toTableOutput(description app.TableDescription) tableOutput {
	out := toEntryOutput(description.Table.Entry)
	out.Source = source
	out.AsOf = instant(description.ReadAt)
	out.Stale = description.Stale
	out.Dimensions = toDimensionsOutput(description.Table.Dimensions)

	return out
}

func toDimensionsOutput(dimensions []domain.Dimension) []dimensionOutput {
	out := make([]dimensionOutput, 0, len(dimensions))

	for _, dimension := range dimensions {
		values := make([]valueOutput, 0, len(dimension.Values))
		for _, value := range dimension.Values {
			values = append(values, valueOutput{Code: value.Code, Text: value.Text})
		}

		out = append(out, dimensionOutput{
			Code: dimension.Code, Text: dimension.Text, Values: values,
		})
	}

	return out
}

func toCellsOutput(cells app.Cells) cellsOutput {
	rows := make([]cellOutput, 0, len(cells.Rows))
	for _, row := range cells.Rows {
		rows = append(rows, cellOutput{Values: row.Values, Figure: figure(row.Figure)})
	}

	return cellsOutput{
		Source:     source,
		AsOf:       instant(cells.ReadAt),
		Stale:      cells.Stale,
		Table:      toEntryOutput(cells.Table.Entry),
		Dimensions: toDimensionsOutput(cells.Table.Dimensions),
		Cells:      rows,
	}
}

func toInflationOutput(result app.InflationResult, language kernel.Language) inflationOutput {
	inflation := result.Inflation

	tables := make([]tableOutput, 0, len(result.Tables))
	for _, entry := range result.Tables {
		tables = append(tables, toEntryOutput(entry))
	}

	return inflationOutput{
		Source:  source,
		AsOf:    instant(result.ReadAt),
		Stale:   result.Stale,
		Month:   inflation.Month.String(),
		Region:  namedOutput{Code: string(inflation.Region), Name: inflation.Region.Name(language)},
		Group:   namedOutput{Code: string(inflation.Group), Name: inflation.Group.Name(language)},
		Annual:  figure(inflation.Annual),
		Monthly: figure(inflation.Monthly),
		Average: figure(inflation.Average),
		Core:    figure(inflation.Core),
		Tables:  tables,
	}
}

// figure writes a published figure as a decimal string, and an unpublished one as null.
func figure(value domain.Figure) *string {
	if !value.Published() {
		return nil
	}

	text := value.String()

	return &text
}

func instant(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
