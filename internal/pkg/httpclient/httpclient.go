// Package httpclient builds the HTTP client that a vendor client uses: a timeout,
// and a limit on the rate of the calls.
package httpclient

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Config is the configuration of the calls to one vendor.
type Config struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s" validate:"gt=0"`
	Rate    float64       `env:"RATE"    envDefault:"5"   validate:"gt=0"`
	Burst   int           `env:"BURST"   envDefault:"5"   validate:"min=1"`
}

// New returns a client that waits for a free slot of the limit before each call.
// A call that cannot get a slot before its deadline fails with
// kernel.ErrSourceUnavailable.
func New(cfg Config) *http.Client {
	return NewWithTransport(cfg, http.DefaultTransport)
}

// NewWithTransport returns the client of New over base. A vendor that refuses the TLS
// handshake of Go needs a base of its own.
func NewWithTransport(cfg Config, base http.RoundTripper) *http.Client {
	return &http.Client{
		Timeout: cfg.Timeout,
		Transport: &limitedTransport{
			base:    base,
			limiter: rate.NewLimiter(rate.Limit(cfg.Rate), cfg.Burst),
		},
	}
}

type limitedTransport struct {
	base    http.RoundTripper
	limiter *rate.Limiter
}

func (t *limitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf(
			"waiting for the vendor limit: %w: %w",
			kernel.ErrSourceUnavailable,
			err,
		)
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("calling the vendor: %w", err)
	}

	return resp, nil
}
