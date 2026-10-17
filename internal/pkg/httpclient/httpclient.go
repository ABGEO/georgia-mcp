// Package httpclient builds the HTTP client that a vendor client uses: a timeout,
// and a limit on the rate of the calls.
package httpclient

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Config is the configuration of the calls to one vendor.
type Config struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"5"`
	Burst   int           `env:"BURST"   envDefault:"5"`
}

// Validate checks that each value is positive.
func (c Config) Validate() error {
	switch {
	case c.Timeout <= 0:
		return &config.FieldError{Field: "Timeout", Problem: "must be more than 0"}
	case c.Rate <= 0:
		return &config.FieldError{Field: "Rate", Problem: "must be more than 0"}
	case c.Burst < 1:
		return &config.FieldError{Field: "Burst", Problem: "must be 1 or more"}
	default:
		return nil
	}
}

// New returns a client that waits for a free slot of the limit before each call.
// A call that cannot get a slot before its deadline fails with
// kernel.ErrSourceUnavailable.
func New(cfg Config) *http.Client {
	return &http.Client{
		Timeout: cfg.Timeout,
		Transport: &limitedTransport{
			base:    http.DefaultTransport,
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
