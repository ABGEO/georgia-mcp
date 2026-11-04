package stats

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	geostatclient "github.com/abgeo/georgia-mcp/pkg/geostat"
)

// A burst of 1 and 9 calls each second keep any second at 10 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second.
const (
	geostatRate  = 9
	geostatBurst = 1
)

// Config is the configuration of the stats context.
type Config struct {
	Geostat vendors.Config `envPrefix:"GEOSTAT_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		Geostat: vendors.Defaults(geostatclient.DefaultBaseURL).
			WithLimit(geostatRate, geostatBurst),
	}
}
