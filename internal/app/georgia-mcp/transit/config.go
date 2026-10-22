package transit

import (
	"net/url"
	"time"
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
	APIKey  string  `env:"API_KEY"  validate:"required_if=Enabled true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://transit.ttc.com.ge/pis-gateway/api" validate:"http_url"`
	HTTP    TTCHTTPConfig
}

// TTCHTTPConfig holds the fields of httpclient.Config with the defaults of the TTC: a
// burst of 1 and 4 calls each second keep any second at 5 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second. A default lives in
// a struct tag, so httpclient.Config cannot carry a second one.
type TTCHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s" validate:"gt=0"`
	Rate    float64       `env:"RATE"    envDefault:"4"   validate:"gt=0"`
	Burst   int           `env:"BURST"   envDefault:"1"   validate:"min=1"`
}
