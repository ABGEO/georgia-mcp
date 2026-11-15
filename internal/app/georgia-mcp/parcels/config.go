package parcels

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	naprclient "github.com/abgeo/georgia-mcp/pkg/napr"
)

// A burst of 5 and 5 calls each second keep any second at 10 calls or fewer, since a
// token bucket lets through at most its burst plus its rate in one second.
const (
	naprRate  = 5
	naprBurst = 5
)

// Config is the configuration of the parcels context.
type Config struct {
	NAPR vendors.Config `envPrefix:"NAPR_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		NAPR: vendors.Defaults(naprclient.DefaultBaseURL).WithLimit(naprRate, naprBurst),
	}
}
