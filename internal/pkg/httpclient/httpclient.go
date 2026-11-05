// Package httpclient builds the HTTP client that a vendor client uses: a timeout, a
// limit on the rate of the calls, and one log record for each call.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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
// kernel.ErrSourceUnavailable. The client writes one record for each call to logger,
// which names the vendor.
func New(cfg Config, logger *slog.Logger) *http.Client {
	return NewWithTransport(cfg, http.DefaultTransport, logger)
}

// NewWithTransport returns the client of New over base. A vendor that refuses the TLS
// handshake of Go needs a base of its own.
func NewWithTransport(cfg Config, base http.RoundTripper, logger *slog.Logger) *http.Client {
	return &http.Client{
		Timeout: cfg.Timeout,
		Transport: &limitedTransport{
			base:    &loggedTransport{base: base, logger: logger},
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

// loggedTransport writes one record for each call that leaves the limit, so the wait for
// a slot is not part of the duration.
type loggedTransport struct {
	base   http.RoundTripper
	logger *slog.Logger
}

func (t *loggedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)

	t.log(req, resp, err, time.Since(start))

	return resp, err //nolint:wrapcheck // limitedTransport wraps the error.
}

func (t *loggedTransport) log(
	req *http.Request,
	resp *http.Response,
	err error,
	duration time.Duration,
) {
	ctx := req.Context()

	attrs := []slog.Attr{
		slog.String("http.request.method", req.Method),
		slog.String("server.address", req.URL.Hostname()),
		slog.Duration("event.duration", duration),
	}

	if t.logger.Enabled(ctx, slog.LevelDebug) {
		attrs = append(attrs, slog.String("url.path", req.URL.Path))
	}

	level := slog.LevelDebug

	switch {
	case err != nil:
		attrs = append(
			attrs,
			slog.String("error.type", fmt.Sprintf("%T", err)),
			slog.Any("error", err),
		)

		if !errors.Is(err, context.Canceled) {
			level = slog.LevelWarn
		}
	case resp.StatusCode >= http.StatusBadRequest:
		attrs = append(attrs,
			slog.Int("http.response.status_code", resp.StatusCode),
			slog.String("error.type", strconv.Itoa(resp.StatusCode)),
		)
		level = statusLevel(resp.StatusCode)
	default:
		attrs = append(attrs, slog.Int("http.response.status_code", resp.StatusCode))
	}

	t.logger.LogAttrs(ctx, level, "vendor request", attrs...)
}

// statusLevel warns on a status that tells that the vendor or its contract fails. A
// wrong argument of the model often gives any other 4xx.
func statusLevel(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError,
		status == http.StatusUnauthorized,
		status == http.StatusForbidden,
		status == http.StatusTooManyRequests:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
