package vendors

import (
	"net/url"

	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
)

// Config is the configuration of a vendor that its context can serve without: the switch
// of the operator, the base address, and the calls. A Config built in Go leaves the
// vendor off. A vendor with a value of its own embeds Config and adds the field.
type Config struct {
	Enabled bool    `env:"ENABLED"`
	BaseURL url.URL `env:"BASE_URL" validate:"http_url"`
	HTTP    httpclient.Config
}

// Defaults returns a vendor that is on, at the base address, with the calls of
// httpclient.Defaults.
func Defaults(baseURL string) Config {
	return Config{Enabled: true, BaseURL: config.URL(baseURL), HTTP: httpclient.Defaults()}
}

// WithLimit returns the configuration with rate calls each second and a burst.
func (c Config) WithLimit(rate float64, burst int) Config {
	c.HTTP = c.HTTP.WithLimit(rate, burst)

	return c
}
