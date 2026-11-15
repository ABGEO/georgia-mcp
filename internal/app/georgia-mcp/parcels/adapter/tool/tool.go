package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// ParcelGetter gives one parcel by its cadastral code.
type ParcelGetter interface {
	Get(ctx context.Context, code domain.CadastralCode) (app.ParcelRead, error)
}

// NearbyLister gives the parcels nearest a point.
type NearbyLister interface {
	List(ctx context.Context, query app.NearbyQuery) (app.NearbyRead, error)
}

// AddressSearcher gives the addresses that match a text.
type AddressSearcher interface {
	Search(ctx context.Context, query app.AddressQuery) (app.AddressList, error)
}

// UseCases holds the use case of each tool.
type UseCases struct {
	GetParcel       ParcelGetter
	ListNearby      NearbyLister
	SearchAddresses AddressSearcher
}

const (
	source = "NAPR"

	defaultLimit = 5
	maxLimit     = 20

	getParcelDescription = "Returns one land parcel of the public registry of Georgia by its " +
		"cadastral code (01.15.05.066.020): the address, the area, the land type, the " +
		"ownership type, the owners, the registry documents with links to their PDFs, and " +
		"the centre point. Without a code, find one with parcels_list_nearby_parcels. NAPR " +
		"publishes every text in Georgian only. Buildings and apartments are not covered. " +
		"Data from NAPR."
	listNearbyDescription = "Returns the land parcels nearest a point in Georgia, nearest " +
		"first, with the cadastral code, the address, and the distance in metres. The first " +
		"parcel at 0 metres holds the point. Get the owners and the area with " +
		"parcels_get_parcel. Addresses are in Georgian. Data from NAPR."
	searchAddressesDescription = "Finds addresses in the address registry of Georgia, with " +
		"the municipality and the centre point of each. Write the street and the number in " +
		"Georgian (ჭავჭავაძის გამზირი 44) and give the municipality (თბილისი): the order of " +
		"NAPR is poor, and other cities often come first. Pass a centre point to " +
		"parcels_list_nearby_parcels to find the parcel. Data from NAPR."
)

// Register adds the tools of the context.
func Register(registry *mcpserver.Registry, uses UseCases) error {
	handlers := handler{uses: uses}

	err := mcpserver.AddTool(registry, &mcp.Tool{
		Name: "parcels_get_parcel", Title: "Get a land parcel of Georgia",
		Description: getParcelDescription,
	}, handlers.getParcel)
	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name:        "parcels_list_nearby_parcels",
			Title:       "List the land parcels near a point in Georgia",
			Description: listNearbyDescription,
		}, handlers.listNearby)
	}

	if err == nil {
		err = mcpserver.AddTool(registry, &mcp.Tool{
			Name: "parcels_search_addresses", Title: "Search the addresses of Georgia",
			Description: searchAddressesDescription,
		}, handlers.searchAddresses)
	}

	if err != nil {
		return fmt.Errorf("registering the parcel tools: %w", err)
	}

	return nil
}

type handler struct {
	uses UseCases
}

func (h handler) getParcel(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in getParcelInput,
) (*mcp.CallToolResult, parcelOutput, error) {
	code, err := parseCode(in.Code)
	if err != nil {
		return nil, parcelOutput{}, err
	}

	read, err := h.uses.GetParcel.Get(ctx, code)
	if err != nil {
		return nil, parcelOutput{}, toolError(err, code)
	}

	return nil, toParcelOutput(read), nil
}

func (h handler) listNearby(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in nearbyInput,
) (*mcp.CallToolResult, nearbyOutput, error) {
	limit, err := withinRange("limit", in.Limit, defaultLimit, maxLimit)
	if err != nil {
		return nil, nearbyOutput{}, err
	}

	point, err := parsePoint(in.Lat, in.Lon)
	if err != nil {
		return nil, nearbyOutput{}, err
	}

	read, err := h.uses.ListNearby.List(ctx, app.NearbyQuery{Point: point, Limit: limit})
	if err != nil {
		return nil, nearbyOutput{}, toolError(err, domain.CadastralCode{})
	}

	return nil, toNearbyOutput(read), nil
}

func (h handler) searchAddresses(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in addressesInput,
) (*mcp.CallToolResult, addressesOutput, error) {
	limit, err := withinRange("limit", in.Limit, defaultLimit, maxLimit)
	if err != nil {
		return nil, addressesOutput{}, err
	}

	if strings.TrimSpace(in.Query) == "" {
		return nil, addressesOutput{}, &mcpserver.ToolError{
			Message: "query: empty. Give a street and a number.",
		}
	}

	list, err := h.uses.SearchAddresses.Search(ctx, app.AddressQuery{
		Text: in.Query, Municipality: in.Municipality, Limit: limit,
	})
	if err != nil {
		return nil, addressesOutput{}, toolError(err, domain.CadastralCode{})
	}

	return nil, toAddressesOutput(list), nil
}
