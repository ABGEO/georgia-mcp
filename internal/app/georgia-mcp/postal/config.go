package postal

import "github.com/abgeo/georgia-mcp/internal/pkg/vendors"

// A burst of 1 and 1 call each second keep any second at 2 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second.
const (
	gpostRate  = 1
	gpostBurst = 1
)

// Config is the configuration of the postal context.
type Config struct {
	GPost vendors.Config `envPrefix:"GPOST_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		GPost: vendors.Defaults("https://www.gpost.ge").WithLimit(gpostRate, gpostBurst),
	}
}
