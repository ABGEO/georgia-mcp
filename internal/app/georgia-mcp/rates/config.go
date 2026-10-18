package rates

import (
	"net/url"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
)

// Config is the configuration of the rates context.
type Config struct {
	NBG NBGConfig `envPrefix:"NBG_"`
}

// NBGConfig is the configuration of the calls to the NBG. HTTP takes the prefix of
// the struct, so its variables are TIMEOUT, RATE, and BURST under NBG_.
type NBGConfig struct {
	BaseURL      url.URL `env:"BASE_URL"      envDefault:"https://nbg.gov.ge/gw/api/ct/monetarypolicy"`
	CacheEntries int     `env:"CACHE_ENTRIES" envDefault:"512"`
	HTTP         httpclient.Config
}

// Validate checks the address and the size of the cache.
func (c NBGConfig) Validate() error {
	if (c.BaseURL.Scheme != "https" && c.BaseURL.Scheme != "http") || c.BaseURL.Host == "" {
		return &config.FieldError{Field: "BaseURL", Problem: "must be an http or https URL"}
	}

	if c.CacheEntries < 1 {
		return &config.FieldError{Field: "CacheEntries", Problem: "must be 1 or more"}
	}

	return nil
}
