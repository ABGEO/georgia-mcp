package transit

import "github.com/abgeo/georgia-mcp/internal/pkg/vendors"

// A burst of 1 and 4 calls each second keep any second at 5 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second.
const (
	ttcRate  = 4
	ttcBurst = 1
)

// Config is the configuration of the transit context.
type Config struct {
	TTC TTCConfig `envPrefix:"TTC_"`
}

// TTCConfig is the configuration of the calls to the TTC. Enabled turns the TTC off. The
// key does not: while the TTC is on, the key is required.
type TTCConfig struct {
	vendors.Config

	APIKey string `env:"API_KEY" validate:"required_if=Enabled true"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{TTC: TTCConfig{
		Config: vendors.Defaults("https://transit.ttc.com.ge/pis-gateway/api").
			WithLimit(ttcRate, ttcBurst),
	}}
}
