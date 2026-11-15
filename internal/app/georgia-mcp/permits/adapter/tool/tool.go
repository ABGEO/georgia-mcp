package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// ApplicationSearcher gives the newest applications of a filter.
type ApplicationSearcher interface {
	Search(ctx context.Context, query app.SearchQuery) (app.ApplicationList, error)
}

// ApplicationGetter gives one application by its number.
type ApplicationGetter interface {
	Get(ctx context.Context, number domain.ApplicationNumber) (app.ApplicationRead, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	SearchApplications ApplicationSearcher
	GetApplication     ApplicationGetter
}

const (
	source = "TAS"

	defaultLimit = 10
	maxLimit     = 50

	searchDescription = "Returns the applications to the Tbilisi Architecture Service for a " +
		"parcel or a period, newest first: building permits, project approvals, signs, " +
		"fences, and requests for public information. Give a cadastral code " +
		"(01.18.04.020.005), a period of at most 31 days, or both. A period of more than " +
		"500 applications fails, and Tbilisi registers about 70 each working day, so ask " +
		"for a week at most. Each application has its number, address, status, category, " +
		"class, and stages. Read the decision with permits_get_application. Tbilisi only. " +
		"Texts are in Georgian. Data from TAS."
	getDescription = "Returns one application to the Tbilisi Architecture Service by its " +
		"number (AR11168188): the address, the status, the category, the class, the " +
		"stages, the deadline, the executor, the text of the decision, and the link to its " +
		"page. Find a number with permits_search_applications. Texts are in Georgian. " +
		"Data from TAS."
)

// Register adds the tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name: "permits_search_applications", Title: "Search the building applications of Tbilisi",
		Description: searchDescription,
	}, handlers.search)
	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "permits_get_application", Title: "Get a building application of Tbilisi",
			Description: getDescription,
		}, handlers.get)
	}

	if err != nil {
		return fmt.Errorf("registering the permit tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) search(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchInput,
) (*mcp.CallToolResult, searchOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, searchOutput{}, err
	}

	limit, err := withinRange("limit", in.Limit, defaultLimit, maxLimit)
	if err != nil {
		return nil, searchOutput{}, err
	}

	filter, err := parseFilter(in)
	if err != nil {
		return nil, searchOutput{}, err
	}

	list, err := h.uses.SearchApplications.Search(
		ctx,
		app.SearchQuery{Filter: filter, Limit: limit},
	)
	if err != nil {
		return nil, searchOutput{}, toolError(err, domain.ApplicationNumber{})
	}

	return nil, toSearchOutput(list, language), nil
}

func (h handler) get(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getInput,
) (*mcp.CallToolResult, getOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, getOutput{}, err
	}

	number, err := parseNumber(in.Number)
	if err != nil {
		return nil, getOutput{}, err
	}

	read, err := h.uses.GetApplication.Get(ctx, number)
	if err != nil {
		return nil, getOutput{}, toolError(err, number)
	}

	return nil, toGetOutput(read, language), nil
}
