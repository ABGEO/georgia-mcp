package energy

import (
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
	gnercclient "github.com/abgeo/georgia-mcp/pkg/gnerc"
	gseclient "github.com/abgeo/georgia-mcp/pkg/gse"
)

// A burst of 2 and 3 calls each second keep any second at 5 calls or fewer, since a token
// bucket lets through at most its burst plus its rate in one second. A burst of 1 and 2
// calls each second keep GNERC at 3.
const (
	gseRate    = 3
	gseBurst   = 2
	gnercRate  = 2
	gnercBurst = 1
)

// Config is the configuration of the energy context.
type Config struct {
	GSE   vendors.Config `envPrefix:"GSE_"`
	GNERC vendors.Config `envPrefix:"GNERC_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		GSE:   vendors.Defaults(gseclient.DefaultBaseURL).WithLimit(gseRate, gseBurst),
		GNERC: vendors.Defaults(gnercclient.DefaultBaseURL).WithLimit(gnercRate, gnercBurst),
	}
}
