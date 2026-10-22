package httpserver

import (
	"time"
)

// Config is the configuration of the HTTP server.
type Config struct {
	Address        string        `env:"ADDRESS"         envDefault:":8080" validate:"hostname_port"`
	TrustedProxies []string      `env:"TRUSTED_PROXIES" envSeparator:","   validate:"dive,cidr"`
	AllowedOrigins []string      `env:"ALLOWED_ORIGINS" envSeparator:","`
	RateLimit      int           `env:"RATE_LIMIT"      envDefault:"60"    validate:"min=1"`
	RateWindow     time.Duration `env:"RATE_WINDOW"     envDefault:"1m"    validate:"gte=1s"`
}
