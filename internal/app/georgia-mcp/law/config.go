package law

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	matsneclient "github.com/abgeo/georgia-mcp/pkg/matsne"
)

// A burst of 5 and 5 calls each second keep any second at 10 calls or fewer, since a
// token bucket lets through at most its burst plus its rate in one second.
const (
	matsneRate  = 5
	matsneBurst = 5
)

// Config is the configuration of the law context.
type Config struct {
	Matsne vendors.Config `envPrefix:"MATSNE_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		Matsne: vendors.Defaults(matsneclient.DefaultBaseURL).WithLimit(matsneRate, matsneBurst),
	}
}
