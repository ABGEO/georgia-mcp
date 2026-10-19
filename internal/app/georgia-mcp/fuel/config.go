package fuel

import (
	"net/url"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
)

// Config is the configuration of the fuel context.
type Config struct {
	Wissol    WissolConfig    `envPrefix:"WISSOL_"`
	Rompetrol RompetrolConfig `envPrefix:"ROMPETROL_"`
	SOCAR     SOCARConfig     `envPrefix:"SOCAR_"`
}

// WissolConfig is the configuration of the calls to Wissol. Wissol needs no key, so
// Enabled turns it off. A Config built in Go leaves Wissol off.
type WissolConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://api.wissol.ge"`
	HTTP    WissolHTTPConfig
}

// Validate checks the address.
func (c WissolConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// WissolHTTPConfig holds the fields of httpclient.Config with the defaults of Wissol: 1
// call each second. Wissol publishes no limit, and the cache sends about 2 calls in an
// hour. A default lives in a struct tag, so httpclient.Config cannot carry a second one.
type WissolHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c WissolHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

// RompetrolConfig is the configuration of the reads of the page of Rompetrol. Rompetrol
// needs no key, so Enabled turns it off. A Config built in Go leaves Rompetrol off.
type RompetrolConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://www.rompetrol.ge"`
	HTTP    RompetrolHTTPConfig
}

// Validate checks the address.
func (c RompetrolConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// RompetrolHTTPConfig holds the fields of httpclient.Config with the defaults of
// Rompetrol: 1 call each second. Rompetrol publishes no limit, and the cache sends about
// 2 calls in an hour.
type RompetrolHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c RompetrolHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

// SOCARConfig is the configuration of the calls to SOCAR. SOCAR needs no key, so Enabled
// turns it off. A Config built in Go leaves SOCAR off.
type SOCARConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://sgp.ge/sgp-backend/api"`
	HTTP    SOCARHTTPConfig
}

// Validate checks the address.
func (c SOCARConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// SOCARHTTPConfig holds the fields of httpclient.Config with the defaults of SOCAR: 1 call
// each second. SOCAR publishes no limit, and the cache sends about 2 calls in an hour.
type SOCARHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c SOCARHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

func validateBaseURL(base url.URL) error {
	if (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" {
		return &config.FieldError{Field: "BaseURL", Problem: "must be an http or https URL"}
	}

	return nil
}
