package crystal

import (
	"context"
	"fmt"
	"net/url"
	"slices"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	// DefaultBaseURL is the production address of the rate app of Crystal.
	DefaultBaseURL = "https://currency.crystal.ge"

	noError = "0"

	// maxErrorText bounds the text of the vendor that an error holds.
	maxErrorText = 200
)

// Client calls the rate method of the rate app of Crystal. It is safe for concurrent use.
type Client struct {
	http *vendorhttp.Client
}

// Option sets one value of a Client.
type Option = vendorhttp.Option

// New returns a client of the rate method of the rate app of Crystal.
func New(opts ...Option) *Client {
	defaults := []Option{
		vendorhttp.WithBaseURL(DefaultBaseURL),
		vendorhttp.WithHeader("Accept", "application/json"),
	}

	return &Client{
		http: vendorhttp.New(ErrUnexpectedStatus, slices.Concat(defaults, opts)...),
	}
}

// CashRates returns each item of the default cash rates, in the order of the answer.
func (c *Client) CashRates(ctx context.Context) ([]Rate, error) {
	var answer envelope

	if err := c.http.FetchJSON(ctx, vendorhttp.Request{
		Path:  []string{"api", "Conversion", "GetWebRateByProductID"},
		Query: url.Values{"IsCash": {"true"}},
	}, &answer); err != nil {
		return nil, fmt.Errorf("getting the cash rates: %w", err)
	}

	if err := answer.check(); err != nil {
		return nil, fmt.Errorf("getting the cash rates: %w", err)
	}

	return answer.Data.Data, nil
}

// check fails when an envelope reports a failure, since the method answers 200 with the
// failure in its body.
func (e envelope) check() error {
	inner := e.Data

	switch {
	case e.ErrorCode != noError:
		return fmt.Errorf("%w: error code %q: %q", ErrNotSucceeded, e.ErrorCode, cut(e.ErrorText))
	case inner == nil:
		return fmt.Errorf("%w: no data", ErrNotSucceeded)
	case !inner.Success || inner.ErrorCode != noError:
		return fmt.Errorf(
			"%w: success %t, error code %q: %q",
			ErrNotSucceeded,
			inner.Success,
			inner.ErrorCode,
			cut(inner.ErrorMessage),
		)
	case inner.Data == nil:
		return fmt.Errorf("%w: no list", ErrNotSucceeded)
	}

	return nil
}

func cut(text string) string {
	if len(text) > maxErrorText {
		return text[:maxErrorText]
	}

	return text
}
