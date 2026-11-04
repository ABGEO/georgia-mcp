package air

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	neaclient "github.com/abgeo/georgia-mcp/pkg/nea"
)

// A burst of 1 and 1 call each second keep any second at 2 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second.
const (
	neaRate  = 1
	neaBurst = 1
)

// Config is the configuration of the air context.
type Config struct {
	NEA vendors.Config `envPrefix:"NEA_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		NEA: vendors.Defaults(neaclient.DefaultBaseURL).WithLimit(neaRate, neaBurst),
	}
}
