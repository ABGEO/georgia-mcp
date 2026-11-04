package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// TableSearcher finds the tables whose title, folder, or code holds every word of a text.
type TableSearcher interface {
	Search(ctx context.Context, query app.TableQuery) (app.TableMatches, error)
}

// TableDescriber gives the title and the dimensions of a table.
type TableDescriber interface {
	Describe(ctx context.Context, request app.TableRequest) (app.TableDescription, error)
}

// CellGetter gives the cells of a table for a selection.
type CellGetter interface {
	Get(ctx context.Context, query app.CellQuery) (app.Cells, error)
}

// InflationGetter gives the inflation of a month.
type InflationGetter interface {
	Get(ctx context.Context, query app.InflationQuery) (app.InflationResult, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	SearchTables  TableSearcher
	DescribeTable TableDescriber
	GetCells      CellGetter
	GetInflation  InflationGetter
}

const (
	defaultTables = 20
	maxTables     = 100

	searchTablesDescription = "Finds the tables of Geostat, the National Statistics Office of " +
		"Georgia, whose title or folder holds every word of a text: for example prices, " +
		"salaries, population, GDP, and trade. Returns the code, the title, the folder, and the " +
		"update date of each table, the newest first. Then call stats_get_table with a code. " +
		"Data from Geostat."
	getTableDescription = "Returns the title and the dimensions of one table of Geostat, with " +
		"the code and the text of each value. Read it before stats_get_cells. The values of a " +
		"time dimension go from the oldest to the newest. A table has no unit field: the title " +
		"names it. Data from Geostat."
	getCellsDescription = "Returns the figures of one table of Geostat for a selection: value " +
		"codes, or the last values, for each dimension. A dimension with no selection takes " +
		"every value. Selects at most 1000 cells. An unpublished figure is null. The title of " +
		"the table names the unit. Data from Geostat."
	getInflationDescription = "Returns the inflation of Georgia for one month, in percent: " +
		"annual (to the same month of the previous year), monthly (to the previous month), the " +
		"12 month average, and core inflation. For the whole country, a city, and a price " +
		"group. The default is the newest published month. Data from the consumer price index " +
		"of Geostat."
)

// Register adds the tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name: "stats_search_tables", Title: "Search the statistical tables of Georgia",
		Description: searchTablesDescription,
	}, handlers.searchTables)
	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "stats_get_table", Title: "Describe a statistical table of Georgia",
			Description: getTableDescription,
		}, handlers.getTable)
	}

	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "stats_get_cells", Title: "Get the figures of a statistical table",
			Description: getCellsDescription,
		}, handlers.getCells)
	}

	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "stats_get_inflation", Title: "Get the inflation of Georgia",
			Description: getInflationDescription,
		}, handlers.getInflation)
	}

	if err != nil {
		return fmt.Errorf("registering the stats tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) searchTables(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchTablesInput,
) (*mcp.CallToolResult, tablesOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, tablesOutput{}, err
	}

	limit, err := withinRange("limit", in.Limit, defaultTables, maxTables)
	if err != nil {
		return nil, tablesOutput{}, err
	}

	matches, err := h.uses.SearchTables.Search(ctx, app.TableQuery{
		Text: in.Query, Language: language, Limit: limit,
	})
	if err != nil {
		return nil, tablesOutput{}, toolError(err, domain.TableCode{})
	}

	return nil, toTablesOutput(matches), nil
}

func (h handler) getTable(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getTableInput,
) (*mcp.CallToolResult, tableOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, tableOutput{}, err
	}

	code, err := parseCode(in.Code)
	if err != nil {
		return nil, tableOutput{}, err
	}

	description, err := h.uses.DescribeTable.Describe(ctx, app.TableRequest{
		Code: code, Language: language,
	})
	if err != nil {
		return nil, tableOutput{}, toolError(err, code)
	}

	return nil, toTableOutput(description), nil
}

func (h handler) getCells(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getCellsInput,
) (*mcp.CallToolResult, cellsOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, cellsOutput{}, err
	}

	code, err := parseCode(in.Code)
	if err != nil {
		return nil, cellsOutput{}, err
	}

	selection := make([]app.DimensionSelection, 0, len(in.Selection))
	for _, item := range in.Selection {
		selection = append(selection, app.DimensionSelection{
			Dimension: item.Dimension, Values: item.Values, Last: item.Last,
		})
	}

	cells, err := h.uses.GetCells.Get(ctx, app.CellQuery{
		Code: code, Selection: selection, Language: language,
	})
	if err != nil {
		return nil, cellsOutput{}, toolError(err, code)
	}

	return nil, toCellsOutput(cells), nil
}

func (h handler) getInflation(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getInflationInput,
) (*mcp.CallToolResult, inflationOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, inflationOutput{}, err
	}

	month, err := parseMonth(in.Month)
	if err != nil {
		return nil, inflationOutput{}, err
	}

	region, err := parseRegion(in.Region)
	if err != nil {
		return nil, inflationOutput{}, err
	}

	group, err := parseGroup(in.Group)
	if err != nil {
		return nil, inflationOutput{}, err
	}

	result, err := h.uses.GetInflation.Get(ctx, app.InflationQuery{
		Month: month, Region: region, Group: group, Language: language,
	})
	if err != nil {
		return nil, inflationOutput{}, toolError(err, domain.TableCode{})
	}

	return nil, toInflationOutput(result, language), nil
}
