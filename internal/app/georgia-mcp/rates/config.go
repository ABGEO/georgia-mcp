package rates

import (
	"net/url"

	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	nbgRate  = 5
	nbgBurst = 5

	// A burst of 10 serves the 10 codes of one request at once, and 0.8 calls each second
	// keep any 60 seconds at 58 calls or fewer, under the 60 that Bank of Georgia allows.
	bogRate  = 0.8
	bogBurst = 10
)

// Config is the configuration of the rates context.
type Config struct {
	NBG      NBGConfig      `envPrefix:"NBG_"`
	TBC      TBCConfig      `envPrefix:"TBC_"`
	BOG      vendors.Config `envPrefix:"BOG_"`
	Liberty  vendors.Config `envPrefix:"LIBERTY_"`
	Terabank vendors.Config `envPrefix:"TERABANK_"`
	Rico     vendors.Config `envPrefix:"RICO_"`
	Kursi    vendors.Config `envPrefix:"KURSI_"`
}

// NBGConfig is the configuration of the calls to the NBG. The context needs the NBG, so it
// has no switch. HTTP takes the prefix of the struct, so its variables are TIMEOUT, RATE,
// and BURST under NBG_.
type NBGConfig struct {
	BaseURL      url.URL `env:"BASE_URL"      envDefault:"https://nbg.gov.ge/gw/api/ct/monetarypolicy" validate:"http_url"`
	CacheEntries int     `env:"CACHE_ENTRIES" envDefault:"512"                                         validate:"min=1"`
	HTTP         httpclient.Config
}

// TBCConfig is the configuration of the calls to TBC Bank. Enabled turns TBC off. The key
// does not: while TBC is on, the key is required.
type TBCConfig struct {
	vendors.Config

	APIKey string `env:"API_KEY" validate:"required_if=Enabled true"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		NBG:      NBGConfig{HTTP: httpclient.Defaults().WithLimit(nbgRate, nbgBurst)},
		TBC:      TBCConfig{Config: vendors.Defaults("https://api.tbcbank.ge")},
		BOG:      vendors.Defaults("https://api.businessonline.ge").WithLimit(bogRate, bogBurst),
		Liberty:  vendors.Defaults("https://www.libertybank.ge"),
		Terabank: vendors.Defaults("https://terabank.ge"),
		Rico:     vendors.Defaults("https://www.rico.ge"),
		Kursi:    vendors.Defaults("https://api-core.kursi.ge"),
	}
}
