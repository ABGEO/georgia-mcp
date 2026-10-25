package cinema

import "github.com/abgeo/georgia-mcp/internal/pkg/vendors"

// A burst of 2 and 2 calls each second keep any second at 4 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second.
const (
	caveaRate  = 2
	caveaBurst = 2
)

// Config is the configuration of the cinema context.
type Config struct {
	Cavea vendors.Config `envPrefix:"CAVEA_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		Cavea: vendors.Defaults("https://cavea.ge").WithLimit(caveaRate, caveaBurst),
	}
}
