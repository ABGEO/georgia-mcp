package transit

import (
	"net/url"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
)

// Config is the configuration of the transit context.
type Config struct {
	TTC TTCConfig `envPrefix:"TTC_"`
}

// TTCConfig is the configuration of the calls to the TTC. Enabled turns the TTC off. The
// key does not: while the TTC is on, the key is required. A Config built in Go leaves the
// TTC off.
type TTCConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	APIKey  string  `env:"API_KEY"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://transit.ttc.com.ge/pis-gateway/api"`
	HTTP    TTCHTTPConfig
}

// Validate refuses an empty key while the TTC is on, and checks the address.
func (c TTCConfig) Validate() error {
	if c.Enabled && c.APIKey == "" {
		return &config.FieldError{
			Field: "APIKey",
			Problem: "must be set while the TTC is on. " +
				"Set it, or set GEOMCP_TRANSIT_TTC_ENABLED=false",
		}
	}

	if (c.BaseURL.Scheme != "https" && c.BaseURL.Scheme != "http") || c.BaseURL.Host == "" {
		return &config.FieldError{Field: "BaseURL", Problem: "must be an http or https URL"}
	}

	return nil
}

// TTCHTTPConfig holds the fields of httpclient.Config with the defaults of the TTC: a
// burst of 1 and 4 calls each second keep any second at 5 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second. A default lives in
// a struct tag, so httpclient.Config cannot carry a second one.
type TTCHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"4"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c TTCHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}
