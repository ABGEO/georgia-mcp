package fuel

import (
	"net/url"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
)

// Config is the configuration of the fuel context.
type Config struct {
	Wissol WissolConfig `envPrefix:"WISSOL_"`
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
	if (c.BaseURL.Scheme != "https" && c.BaseURL.Scheme != "http") || c.BaseURL.Host == "" {
		return &config.FieldError{Field: "BaseURL", Problem: "must be an http or https URL"}
	}

	return nil
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
