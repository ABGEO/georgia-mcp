package fuel

import "github.com/abgeo/georgia-mcp/internal/pkg/vendors"

// Config is the configuration of the fuel context: one vendor for each fuel company.
type Config struct {
	Wissol    vendors.Config `envPrefix:"WISSOL_"`
	Rompetrol vendors.Config `envPrefix:"ROMPETROL_"`
	SOCAR     vendors.Config `envPrefix:"SOCAR_"`
	Connect   vendors.Config `envPrefix:"CONNECT_"`
	Lukoil    vendors.Config `envPrefix:"LUKOIL_"`
	Neogas    vendors.Config `envPrefix:"NEOGAS_"`
	Portal    vendors.Config `envPrefix:"PORTAL_"`
	Gulf      vendors.Config `envPrefix:"GULF_"`
}

// Defaults returns the configuration of the context with no variable set.
func Defaults() Config {
	return Config{
		Wissol:    vendors.Defaults("https://api.wissol.ge"),
		Rompetrol: vendors.Defaults("https://www.rompetrol.ge"),
		SOCAR:     vendors.Defaults("https://sgp.ge/sgp-backend/api"),
		Connect:   vendors.Defaults("https://connect.com.ge/wp-json/fuel-prices/v1"),
		Lukoil:    vendors.Defaults("https://www.lukoil.ge"),
		Neogas:    vendors.Defaults("https://neogas.ge"),
		Portal:    vendors.Defaults("https://portal.com.ge"),
		Gulf:      vendors.Defaults("https://gulf.ge"),
	}
}
