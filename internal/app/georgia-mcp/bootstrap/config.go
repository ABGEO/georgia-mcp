package bootstrap

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/config"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/logger"
)

// Config is the configuration of the process. A context adds its own struct here.
type Config struct {
	Log  logger.Config     `envPrefix:"LOG_"`
	HTTP httpserver.Config `envPrefix:"HTTP_"`
}

// LoadConfig reads and validates the configuration.
func LoadConfig() (Config, error) {
	return config.Load[Config]()
}
