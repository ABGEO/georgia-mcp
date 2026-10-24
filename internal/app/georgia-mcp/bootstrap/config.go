package bootstrap

import (
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit"
	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/logger"
)

// Config is the configuration of the process. A context adds its own struct here.
type Config struct {
	Log     logger.Config     `envPrefix:"LOG_"`
	HTTP    httpserver.Config `envPrefix:"HTTP_"`
	Rates   rates.Config      `envPrefix:"RATES_"`
	Fuel    fuel.Config       `envPrefix:"FUEL_"`
	Transit transit.Config    `envPrefix:"TRANSIT_"`
	Offers  offers.Config     `envPrefix:"OFFERS_"`
	Postal  postal.Config     `envPrefix:"POSTAL_"`
}

// LoadConfig reads and validates the configuration.
func LoadConfig() (Config, error) {
	return config.Load(Defaults())
}

// Defaults returns the configuration of the process with no variable set.
func Defaults() Config {
	return Config{
		Rates:   rates.Defaults(),
		Fuel:    fuel.Defaults(),
		Transit: transit.Defaults(),
		Offers:  offers.Defaults(),
		Postal:  postal.Defaults(),
	}
}
