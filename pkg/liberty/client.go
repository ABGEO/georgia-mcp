package liberty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the site of Studio.
	DefaultBaseURL = "https://studioliberty.ge"

	vendorName = "liberty"
	apiPath    = "_mvcapi/offersapi"

	// maxBodyBytes bounds the read of an answer. The page of an offer is under 60 KiB.
	maxBodyBytes = 2 << 20
)

// Client calls the offers API of Studio and reads the page of an offer. It is safe for
// concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the offers of Studio.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithMaxBody(maxBodyBytes),
	}

	return &Client{
		http: vendorhttp.New(vendorName, ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// answer is the wrapper of each answer of the API.
type answer[T any] struct {
	Data struct {
		List []T `json:"list"`
	} `json:"data"`
	Succeeded bool   `json:"succeeded"`
	Message   string `json:"message"`
}

type languageBody struct {
	LangAbbr string `json:"langAbbr"`
}

type offersBody struct {
	CategoryID    *int   `json:"categoryId"`
	SubCategoryID *int   `json:"subCategoryId"`
	TypeID        *int   `json:"typeId"`
	PartnerID     *int   `json:"partnerId"`
	LocationID    *int   `json:"locationId"`
	SortBy        string `json:"sortBy"`
	Page          int    `json:"page"`
	LangAbbr      string `json:"langAbbr"`
}

// Categories returns the categories of the offers in a language.
func (c *Client) Categories(ctx context.Context, language string) ([]Category, error) {
	categories, err := call[Category](ctx, c, "loadcategories", languageBody{LangAbbr: language})
	if err != nil {
		return nil, fmt.Errorf("getting the categories: %w", err)
	}

	return categories, nil
}

// Types returns the types of the offers in a language.
func (c *Client) Types(ctx context.Context, language string) ([]Type, error) {
	types, err := call[Type](ctx, c, "loadtypes", languageBody{LangAbbr: language})
	if err != nil {
		return nil, fmt.Errorf("getting the types: %w", err)
	}

	return types, nil
}

// Offers returns one page of at most 16 offers. A page with less than 16 is the last.
func (c *Client) Offers(ctx context.Context, query OffersQuery) ([]Offer, error) {
	body := offersBody{
		CategoryID: idOrNull(query.CategoryID),
		TypeID:     idOrNull(query.TypeID),
		Page:       query.Page,
		LangAbbr:   query.Language,
	}

	offers, err := call[Offer](ctx, c, "loadOffers", body)
	if err != nil {
		return nil, fmt.Errorf("getting page %d of the offers: %w", query.Page, err)
	}

	return offers, nil
}

// OfferPage reads the page of one offer in a language.
func (c *Client) OfferPage(ctx context.Context, id int, language string) (Page, error) {
	body, err := c.http.Fetch(ctx, vendorhttp.Request{
		Path: []string{
			language, "studio-lifestyle", "offers", "details", strconv.Itoa(id),
		},
		Header: http.Header{"Accept": {"text/html"}},
	})
	if err != nil {
		return Page{}, fmt.Errorf("getting the page of the offer %d: %w", id, err)
	}

	page, err := parsePage(body)
	if err != nil {
		return Page{}, fmt.Errorf("reading the page of the offer %d: %w", id, err)
	}

	return page, nil
}

func idOrNull(id int) *int {
	if id == 0 {
		return nil
	}

	return &id
}

// call posts a JSON body to a method of the API and returns the list of a success.
func call[T any](ctx context.Context, client *Client, method string, request any) ([]T, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encoding the request: %w", err)
	}

	var decoded answer[T]

	err = client.http.FetchJSON(ctx, vendorhttp.Request{
		Method: http.MethodPost,
		Path:   []string{apiPath, method},
		Header: http.Header{
			"Accept":       {"application/json"},
			"Content-Type": {"application/json"},
		},
		Body: payload,
	}, &decoded)
	if err != nil {
		return nil, err
	}

	if !decoded.Succeeded {
		return nil, fmt.Errorf("%w: %q", ErrNoSuccess, decoded.Message)
	}

	return decoded.Data.List, nil
}
