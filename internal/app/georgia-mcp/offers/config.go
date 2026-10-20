package offers

import (
	"net/url"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
)

// Config is the configuration of the offers context.
type Config struct {
	BOG BOGConfig `envPrefix:"BOG_"`
	TBC TBCConfig `envPrefix:"TBC_"`
}

// BOGConfig is the configuration of the calls to Bank of Georgia. BOG needs no key, so
// Enabled turns it off. A Config built in Go leaves BOG off.
type BOGConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://bankofgeorgia.ge/api/bog-b/offers-hub"`
	HTTP    BOGHTTPConfig
}

// Validate checks the address.
func (c BOGConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// BOGHTTPConfig holds the fields of httpclient.Config with the defaults of BOG: 1 call
// each second. A read of BOG is 3 calls, and the cache sends one read each hour for each
// language.
type BOGHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c BOGHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

// TBCConfig is the configuration of the calls to TBC Bank. TBC needs no key, so Enabled
// turns it off. A Config built in Go leaves TBC off.
type TBCConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://apigw.tbcbank.ge/api/v1/marketing"`
	HTTP    TBCHTTPConfig
}

// Validate checks the address.
func (c TBCConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// TBCHTTPConfig holds the fields of httpclient.Config with the defaults of TBC: 5 calls
// each second. A read of TBC is about 40 calls, one for each filter.
type TBCHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"5"`
	Burst   int           `env:"BURST"   envDefault:"5"`
}

// Validate checks each value as httpclient.Config does.
func (c TBCHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

func validateBaseURL(base url.URL) error {
	if (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" {
		return &config.FieldError{Field: "BaseURL", Problem: "must be an http or https URL"}
	}

	return nil
}
