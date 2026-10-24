package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// CodeSearcher finds the postal codes of the addresses that match a text.
type CodeSearcher interface {
	Search(ctx context.Context, query app.CodeQuery) (app.Codes, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	SearchCodes CodeSearcher
}

const (
	source = "Georgian Post"

	defaultAddresses = 20
	maxAddresses     = 100

	searchCodesDescription = "Finds the postal codes of Georgian Post for an address, a street, " +
		"a town, or a village of Georgia, by text in Georgian or English. Returns each matching " +
		"address with its 4-digit postal code and its district. Georgian Post gives a postal code " +
		"for each street in Tbilisi, Batumi, and Kutaisi only. For another town or a village, " +
		"search its name. Data from the web page of Georgian Post."
	emptyNote = "Georgian Post found no address. In Tbilisi, Batumi, and Kutaisi, give a " +
		"street. For another town or a village, give only its name, for example Rustavi."
)

// Register adds the tool of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name: "postal_search_codes", Title: "Search the postal codes of Georgia",
		Description: searchCodesDescription,
	}, handlers.searchCodes)
	if err != nil {
		return fmt.Errorf("registering the postal tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) searchCodes(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in searchCodesInput,
) (*mcp.CallToolResult, codesOutput, error) {
	language, err := parseLanguage(in.Language)
	if err != nil {
		return nil, codesOutput{}, err
	}

	count, err := withinRange("limit", in.Limit, defaultAddresses, maxAddresses)
	if err != nil {
		return nil, codesOutput{}, err
	}

	codes, err := h.uses.SearchCodes.Search(ctx, app.CodeQuery{
		Text: in.Query, Language: language, Limit: count,
	})
	if err != nil {
		return nil, codesOutput{}, toolError(err)
	}

	return nil, toOutput(codes), nil
}
