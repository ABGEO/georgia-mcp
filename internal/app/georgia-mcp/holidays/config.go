package holidays

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	nagerclient "github.com/abgeo/georgia-mcp/pkg/nager"
)

// A burst of 1 and 1 call each second keep any second at 2 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second.
const (
	nagerRate  = 1
	nagerBurst = 1
)

// Config is the configuration of the holidays context.
type Config struct {
	Nager vendors.Config `envPrefix:"NAGER_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		Nager: vendors.Defaults(nagerclient.DefaultBaseURL).WithLimit(nagerRate, nagerBurst),
	}
}
