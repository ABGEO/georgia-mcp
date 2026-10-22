package offers

import (
	"net/url"
	"time"
)

// Config is the configuration of the offers context.
type Config struct {
	BOG     BOGConfig     `envPrefix:"BOG_"`
	TBC     TBCConfig     `envPrefix:"TBC_"`
	Liberty LibertyConfig `envPrefix:"LIBERTY_"`
}

// BOGConfig is the configuration of the calls to Bank of Georgia. BOG needs no key, so
// Enabled turns it off. A Config built in Go leaves BOG off.
type BOGConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://bankofgeorgia.ge/api/bog-b/offers-hub" validate:"http_url"`
	HTTP    BOGHTTPConfig
}

// BOGHTTPConfig holds the fields of httpclient.Config with the defaults of BOG: 1 call
// each second. A read of BOG is 3 calls, and the cache sends one read each hour for each
// language.
type BOGHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s" validate:"gt=0"`
	Rate    float64       `env:"RATE"    envDefault:"1"   validate:"gt=0"`
	Burst   int           `env:"BURST"   envDefault:"1"   validate:"min=1"`
}

// TBCConfig is the configuration of the calls to TBC Bank. TBC needs no key, so Enabled
// turns it off. A Config built in Go leaves TBC off.
type TBCConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://apigw.tbcbank.ge/api/v1/marketing" validate:"http_url"`
	HTTP    TBCHTTPConfig
}

// TBCHTTPConfig holds the fields of httpclient.Config with the defaults of TBC: 5 calls
// each second. A read of TBC is about 40 calls, one for each filter.
type TBCHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s" validate:"gt=0"`
	Rate    float64       `env:"RATE"    envDefault:"5"   validate:"gt=0"`
	Burst   int           `env:"BURST"   envDefault:"5"   validate:"min=1"`
}

// LibertyConfig is the configuration of the calls to Liberty. Liberty needs no key, so
// Enabled turns it off. A Config built in Go leaves Liberty off.
type LibertyConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://studioliberty.ge" validate:"http_url"`
	HTTP    LibertyHTTPConfig
}

// LibertyHTTPConfig holds the fields of httpclient.Config with the defaults of Liberty:
// 5 calls each second. A read of Liberty is about 67 calls, one for each page of a list.
type LibertyHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s" validate:"gt=0"`
	Rate    float64       `env:"RATE"    envDefault:"5"   validate:"gt=0"`
	Burst   int           `env:"BURST"   envDefault:"5"   validate:"min=1"`
}
