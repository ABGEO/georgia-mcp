package offers

import "github.com/abgeo/georgia-mcp/internal/pkg/vendors"

// A read of TBC is about 40 calls, one for each filter, and a read of Liberty about 67,
// one for each page of a list: both banks take 5 calls each second.
const (
	gatewayRate  = 5
	gatewayBurst = 5
)

// Config is the configuration of the offers context: one vendor for each bank.
type Config struct {
	BOG     vendors.Config `envPrefix:"BOG_"`
	TBC     vendors.Config `envPrefix:"TBC_"`
	Liberty vendors.Config `envPrefix:"LIBERTY_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		BOG: vendors.Defaults("https://bankofgeorgia.ge/api/bog-b/offers-hub"),
		TBC: vendors.Defaults("https://apigw.tbcbank.ge/api/v1/marketing").
			WithLimit(gatewayRate, gatewayBurst),
		Liberty: vendors.Defaults("https://studioliberty.ge").WithLimit(gatewayRate, gatewayBurst),
	}
}
