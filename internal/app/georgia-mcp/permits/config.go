package permits

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	tasclient "github.com/abgeo/georgia-mcp/pkg/tas"
)

// A burst of 1 and 1 call each second keep any second at 2 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second.
const (
	tasRate  = 1
	tasBurst = 1
)

// Config is the configuration of the permits context.
type Config struct {
	TAS vendors.Config `envPrefix:"TAS_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		TAS: vendors.Defaults(tasclient.DefaultBaseURL).WithLimit(tasRate, tasBurst),
	}
}
