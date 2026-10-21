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
	Connect   ConnectConfig   `envPrefix:"CONNECT_"`
	Lukoil    LukoilConfig    `envPrefix:"LUKOIL_"`
	Neogas    NeogasConfig    `envPrefix:"NEOGAS_"`
	Portal    PortalConfig    `envPrefix:"PORTAL_"`
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

// ConnectConfig is the configuration of the calls to Connect. Connect needs no key, so
// Enabled turns it off. A Config built in Go leaves Connect off.
type ConnectConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://connect.com.ge/wp-json/fuel-prices/v1"`
	HTTP    ConnectHTTPConfig
}

// Validate checks the address.
func (c ConnectConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// ConnectHTTPConfig holds the fields of httpclient.Config with the defaults of Connect: 1
// call each second. Connect publishes no limit, and the cache sends about 2 calls in an hour.
type ConnectHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c ConnectHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

// LukoilConfig is the configuration of the reads of the price history page of Lukoil. Lukoil needs no key, so
// Enabled turns it off. A Config built in Go leaves Lukoil off.
type LukoilConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://www.lukoil.ge"`
	HTTP    LukoilHTTPConfig
}

// Validate checks the address.
func (c LukoilConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// LukoilHTTPConfig holds the fields of httpclient.Config with the defaults of Lukoil: 1
// call each second. Lukoil publishes no limit, and the cache sends about 2 calls in an hour.
type LukoilHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c LukoilHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

// NeogasConfig is the configuration of the reads of the home page of Neogas. Neogas needs no key, so
// Enabled turns it off. A Config built in Go leaves Neogas off.
type NeogasConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://neogas.ge"`
	HTTP    NeogasHTTPConfig
}

// Validate checks the address.
func (c NeogasConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// NeogasHTTPConfig holds the fields of httpclient.Config with the defaults of Neogas: 1
// call each second. Neogas publishes no limit, and the cache sends about 2 calls in an hour.
type NeogasHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c NeogasHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

// PortalConfig is the configuration of the reads of the price page of Portal. Portal needs no key, so
// Enabled turns it off. A Config built in Go leaves Portal off.
type PortalConfig struct {
	Enabled bool    `env:"ENABLED"  envDefault:"true"`
	BaseURL url.URL `env:"BASE_URL" envDefault:"https://portal.com.ge"`
	HTTP    PortalHTTPConfig
}

// Validate checks the address.
func (c PortalConfig) Validate() error {
	return validateBaseURL(c.BaseURL)
}

// PortalHTTPConfig holds the fields of httpclient.Config with the defaults of Portal: 1
// call each second. Portal publishes no limit, and the cache sends about 2 calls in an hour.
type PortalHTTPConfig struct {
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s"`
	Rate    float64       `env:"RATE"    envDefault:"1"`
	Burst   int           `env:"BURST"   envDefault:"1"`
}

// Validate checks each value as httpclient.Config does.
func (c PortalHTTPConfig) Validate() error {
	return httpclient.Config(c).Validate()
}

func validateBaseURL(base url.URL) error {
	if (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" {
		return &config.FieldError{Field: "BaseURL", Problem: "must be an http or https URL"}
	}

	return nil
}
