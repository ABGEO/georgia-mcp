package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// GridGetter gives the grid state of the newest slot, or of the slot of a moment.
type GridGetter interface {
	Get(ctx context.Context, query app.GridQuery) (app.GridRead, error)
}

// BalanceGetter gives the balance of a month or a year.
type BalanceGetter interface {
	Get(ctx context.Context, query app.BalanceQuery) (app.BalanceAnswer, error)
}

// PriceLister gives the prices of a period.
type PriceLister interface {
	List(ctx context.Context, query app.PriceQuery) (app.PriceAnswer, error)
}

// UseCases holds the use case of each tool. A nil use case adds no tool: its vendor is off.
type UseCases struct {
	GetGrid    GridGetter
	GetBalance BalanceGetter
	ListPrices PriceLister
}

const (
	source = "Georgian State Electrosystem (GSE)"

	getGridDescription = "Returns the state of the electricity grid of Georgia from Georgian " +
		"State Electrosystem (GSE), the operator of the transmission grid: the generation of " +
		"the hydro, thermal, wind, and solar power plants, the total generation, the " +
		"consumption, and the flow on the lines to each neighbor country, in MW, with the " +
		"frequency in Hz. Each value is the average of 3 minutes. A positive flow is an " +
		"import, a negative flow an export. Each value carries the quality code of GSE: only " +
		"ACT is a measurement. Give a moment to read the grid at a time in the last 31 days. " +
		"The names are in English. Data from GSE."
)

// Register adds the tool of each use case that UseCases holds.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	if uses.GetGrid != nil {
		if err := mcpserver.AddTool(registry, &mcp.Tool{
			Name: "energy_get_grid", Title: "Get the electricity grid of Georgia",
			Description: getGridDescription,
		}, handlers.getGrid); err != nil {
			return fmt.Errorf("registering the energy tools: %w", err)
		}
	}

	if uses.GetBalance != nil {
		if err := mcpserver.AddTool(registry, &mcp.Tool{
			Name: "energy_get_balance", Title: "Get the electricity balance of Georgia",
			Description: getBalanceDescription,
		}, handlers.getBalance); err != nil {
			return fmt.Errorf("registering the energy tools: %w", err)
		}
	}

	if uses.ListPrices != nil {
		if err := mcpserver.AddTool(registry, &mcp.Tool{
			Name: "energy_list_prices", Title: "List the electricity prices of Georgia",
			Description: listPricesDescription,
		}, handlers.listPrices); err != nil {
			return fmt.Errorf("registering the energy tools: %w", err)
		}
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) getGrid(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getGridInput,
) (*mcp.CallToolResult, gridOutput, error) {
	moment, err := parseMoment(in.Moment)
	if err != nil {
		return nil, gridOutput{}, err
	}

	read, err := h.uses.GetGrid.Get(ctx, app.GridQuery{Moment: moment})
	if err != nil {
		return nil, gridOutput{}, toolError(err)
	}

	return nil, toGridOutput(read), nil
}
