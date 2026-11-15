package parliament

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	parliamentclient "github.com/abgeo/georgia-mcp/pkg/parliament"
)

// A bill sends at most about 6 calls, so 5 calls each second with a burst of 5 never make
// one request wait.
const (
	parliamentRate  = 5
	parliamentBurst = 5
)

// Config is the configuration of the parliament context.
type Config struct {
	Parliament vendors.Config `envPrefix:"PARLIAMENT_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		Parliament: vendors.Defaults(parliamentclient.DefaultBaseURL).
			WithLimit(parliamentRate, parliamentBurst),
	}
}
