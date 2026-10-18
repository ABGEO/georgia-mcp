package rates

import (
	"net/url"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
)

// Config is the configuration of the rates context.
type Config struct {
	NBG NBGConfig `envPrefix:"NBG_"`
	TBC TBCConfig `envPrefix:"TBC_"`
	BOG BOGConfig `envPrefix:"BOG_"`
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
	if err := validateBaseURL(c.BaseURL); err != nil {
		return err
	}

	if c.CacheEntries < 1 {
		return &config.FieldError{Field: "CacheEntries", Problem: "must be 1 or more"}
	}

	return nil
}

// TBCConfig is the configuration of the calls to TBC Bank. An empty APIKey turns TBC off.
type TBCConfig struct {
	APIKey  string  `env:"API_KEY"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://api.tbcbank.ge"`
	HTTP    TBCHTTPConfig
}

// Validate checks the address.
func (c TBCConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// TBCHTTPConfig holds the fields of httpclient.Config with the defaults of TBC: 1 call
// each second, under the 60 calls in a minute that TBC allows for each key. A default
// lives in a struct tag, so httpclient.Config cannot carry a second one.
type TBCHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c TBCHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

// BOGConfig is the configuration of the calls to Bank of Georgia. The bank needs no key,
// so Enabled turns it off. A Config built in Go leaves the bank off.
type BOGConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://api.businessonline.ge"`
	HTTP    BOGHTTPConfig
}

// Validate checks the address.
func (c BOGConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// BOGHTTPConfig holds the fields of httpclient.Config with the defaults of Bank of
// Georgia: a burst of 10 serves the 10 codes of one request at once, and 0.8 calls each
// second keep any 60 seconds at 58 calls or fewer, under the 60 that the bank allows.
type BOGHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"0.8"`
	Burst   int           `env:"BURST"   envDefault:"10"`
}

// Validate checks each value as httpclient.Config does.
func (c BOGHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

func validateBaseURL(baseURL url.URL) error {
	if (baseURL.Scheme != "https" && baseURL.Scheme != "http") || baseURL.Host == "" {
		return &config.FieldError{Field: "BaseURL", Problem: "must be an http or https URL"}
	}

	return nil
}
