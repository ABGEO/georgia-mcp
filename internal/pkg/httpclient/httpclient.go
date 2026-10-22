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

const (
	defaultTimeout = 10 * time.Second
	defaultRate    = 1
	defaultBurst   = 1
)

// Config is the configuration of the calls to one vendor.
type Config struct {
	Timeout time.Duration `env:"TIMEOUT" validate:"gt=0"`
	Rate    float64       `env:"RATE"    validate:"gt=0"`
	Burst   int           `env:"BURST"   validate:"min=1"`
}

// Defaults returns the calls of most vendors: a timeout of 10 seconds, and 1 call each
// second with a burst of 1.
func Defaults() Config {
	return Config{Timeout: defaultTimeout, Rate: defaultRate, Burst: defaultBurst}
}

// WithLimit returns the configuration with rate calls each second and a burst.
func (c Config) WithLimit(rate float64, burst int) Config {
	c.Rate, c.Burst = rate, burst

	return c
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
