package vendorhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"
)

const (
	// DefaultTimeout is the timeout of the default HTTP client.
	DefaultTimeout = 10 * time.Second

	defaultMaxBody = 1 << 20
	maxErrorBody   = 1 << 10
	repositoryURL  = "https://github.com/abgeo/georgia-mcp"
)

// Client sends the requests of one vendor client. It is safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	header     http.Header
	maxBody    int64
	sentinel   error
}

// Option sets one value of a Client.
type Option func(*Client)

// WithBaseURL sets the address of the vendor.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets the HTTP client. The default has a timeout of DefaultTimeout.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithUserAgent sets the User-Agent header. The default is
// georgia-mcp/<version> (+https://github.com/abgeo/georgia-mcp).
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.userAgent = userAgent }
}

// WithHeader sets a header that each request sends, for example Accept or a key.
func WithHeader(key, value string) Option {
	return func(c *Client) { c.header.Set(key, value) }
}

// WithMaxBody sets the count of bytes that the client reads at most from an answer. The
// default is 1 MiB.
func WithMaxBody(maxBody int64) Option {
	return func(c *Client) { c.maxBody = maxBody }
}

// New returns a client for a vendor client. A status error wraps sentinel.
func New(sentinel error, opts ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		userAgent:  userAgent(),
		header:     http.Header{},
		maxBody:    defaultMaxBody,
		sentinel:   sentinel,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Request is one call to the vendor.
type Request struct {
	// Method is GET when it is empty.
	Method string
	// Path holds the segments after the base address.
	Path  []string
	Query url.Values
	// Header wins over the header of the client.
	Header http.Header
	Body   []byte
}

// Response is an answer of the vendor, with at most the limit of the body.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// ErrorBody returns the first 1 KiB of the body.
func (r *Response) ErrorBody() string {
	return string(r.Body[:min(len(r.Body), maxErrorBody)])
}

// Do sends the request and returns the answer of any status.
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	endpoint, err := url.JoinPath(c.baseURL, req.Path...)
	if err != nil {
		return nil, fmt.Errorf("building the address: %w", err)
	}

	if len(req.Query) > 0 {
		endpoint += "?" + req.Query.Encode()
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader = http.NoBody
	if req.Body != nil {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("building the request: %w", err)
	}

	httpReq.Header = c.header.Clone()

	for key, values := range req.Header {
		httpReq.Header[http.CanonicalHeaderKey(key)] = values
	}

	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending the request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	read, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBody))
	if err != nil {
		return nil, fmt.Errorf("reading the answer: %w", err)
	}

	return &Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: read}, nil
}

// Fetch returns the body of a 200. Any other status is a *StatusError.
func (c *Client) Fetch(ctx context.Context, req Request) ([]byte, error) {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.StatusError(resp)
	}

	return resp.Body, nil
}

// FetchJSON decodes the body of Fetch into target.
func (c *Client) FetchJSON(ctx context.Context, req Request, target any) error {
	body, err := c.Fetch(ctx, req)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decoding the answer: %w", err)
	}

	return nil
}

// StatusError returns the error of an answer whose status is not a success.
func (c *Client) StatusError(resp *Response) *StatusError {
	return &StatusError{StatusCode: resp.StatusCode, Body: resp.ErrorBody(), sentinel: c.sentinel}
}

// StatusError is an answer of a vendor with a status that is not a success.
type StatusError struct {
	StatusCode int
	// Body is the first 1 KiB of the answer.
	Body     string
	sentinel error
}

// Error returns the sentinel and the status.
func (e *StatusError) Error() string {
	return fmt.Sprintf("%s: %d: %s", e.sentinel, e.StatusCode, http.StatusText(e.StatusCode))
}

// Unwrap returns the sentinel of the client.
func (e *StatusError) Unwrap() error {
	return e.sentinel
}

// userAgent returns georgia-mcp/<the version of the server> (+<repository>).
func userAgent() string {
	return "georgia-mcp/" + version() + " (+" + repositoryURL + ")"
}

func version() string {
	build, ok := debug.ReadBuildInfo()
	if !ok || build.Main.Version == "" || build.Main.Version == "(devel)" {
		return "dev"
	}

	return build.Main.Version
}
