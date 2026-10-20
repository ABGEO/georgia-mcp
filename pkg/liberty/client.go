package liberty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	// DefaultBaseURL is the production address of the site of Studio.
	DefaultBaseURL = "https://studioliberty.ge"

	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
	modulePath     = "github.com/abgeo/georgia-mcp/pkg/liberty"
	defaultTimeout = 10 * time.Second
	apiPath        = "_mvcapi/offersapi"

	// maxBodyBytes bounds the read of an answer. The page of an offer is under 60 KiB.
	maxBodyBytes = 2 << 20
	maxErrorBody = 1 << 10
)

// Client calls the offers API of Studio and reads the page of an offer. It is safe for
// concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// Option sets one value of a Client.
type Option func(*Client)

// WithBaseURL sets the address of the site. The default is DefaultBaseURL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets the HTTP client. The default has a timeout of 10 seconds.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithUserAgent sets the User-Agent header. The default is
// georgia-mcp-liberty/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// New returns a client of the offers of Studio.
func New(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "georgia-mcp-liberty/" + version() + " (+" + repositoryURL + ")",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
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
	endpoint, err := url.JoinPath(
		c.baseURL,
		language,
		"studio-lifestyle",
		"offers",
		"details",
		strconv.Itoa(id),
	)
	if err != nil {
		return Page{}, fmt.Errorf("building the address: %w", err)
	}

	body, err := c.send(ctx, http.MethodGet, endpoint, nil, "text/html")
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
	endpoint, err := url.JoinPath(client.baseURL, apiPath, method)
	if err != nil {
		return nil, fmt.Errorf("building the address: %w", err)
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encoding the request: %w", err)
	}

	body, err := client.send(ctx, http.MethodPost, endpoint, payload, "application/json")
	if err != nil {
		return nil, err
	}

	var decoded answer[T]
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decoding the answer: %w", err)
	}

	if !decoded.Succeeded {
		return nil, fmt.Errorf("%w: %q", ErrNoSuccess, decoded.Message)
	}

	return decoded.Data.List, nil
}

// send sends a request, with a JSON body when payload is not nil, and returns the body
// of a 200. Any other status is an *APIError.
func (c *Client) send(
	ctx context.Context,
	method, endpoint string,
	payload []byte,
	accept string,
) ([]byte, error) {
	var reader io.Reader = http.NoBody
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}

	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", c.userAgent)

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending the request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reading the answer: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body[:min(len(body), maxErrorBody)]),
		}
	}

	return body, nil
}

func version() string {
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	if build.Main.Path == modulePath && build.Main.Version != "" {
		return build.Main.Version
	}

	for _, dep := range build.Deps {
		if dep.Path == modulePath {
			return dep.Version
		}
	}

	return "dev"
}
