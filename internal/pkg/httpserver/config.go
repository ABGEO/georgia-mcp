package httpserver

import (
	"net"
	"net/netip"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
)

const minRateWindow = time.Second

// Config is the configuration of the HTTP server.
type Config struct {
	Address        string        `env:"ADDRESS"         envDefault:":8080"`
	TrustedProxies []string      `env:"TRUSTED_PROXIES" envSeparator:","`
	AllowedOrigins []string      `env:"ALLOWED_ORIGINS" envSeparator:","`
	RateLimit      int           `env:"RATE_LIMIT"      envDefault:"60"`
	RateWindow     time.Duration `env:"RATE_WINDOW"     envDefault:"1m"`
}

// Validate checks the address, the proxy ranges, and the rate limit.
func (c Config) Validate() error {
	if _, _, err := net.SplitHostPort(c.Address); err != nil {
		return &config.FieldError{Field: "Address", Problem: "invalid address, want host:port"}
	}

	for _, prefix := range c.TrustedProxies {
		if _, err := netip.ParsePrefix(prefix); err != nil {
			return &config.FieldError{
				Field:   "TrustedProxies",
				Problem: "invalid range " + prefix + ", want a CIDR",
			}
		}
	}

	if c.RateLimit < 1 {
		return &config.FieldError{Field: "RateLimit", Problem: "must be 1 or more"}
	}

	if c.RateWindow < minRateWindow {
		return &config.FieldError{Field: "RateWindow", Problem: "must be 1s or more"}
	}

	return nil
}
