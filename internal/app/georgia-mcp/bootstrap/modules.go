package bootstrap

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

// builder builds one bounded context.
type builder struct {
	name  string
	build func() (mcpserver.Module, error)
}

// contextOf adapts the constructor of a context to a builder.
func contextOf[C any, M mcpserver.Module](
	name string,
	newModule func(C, *slog.Logger) (M, error),
	cfg C,
	logger *slog.Logger,
) builder {
	return builder{name: name, build: func() (mcpserver.Module, error) {
		return newModule(cfg, logger)
	}}
}

// modules builds every bounded context, in order. A new context adds one line here. When
// a context fails, the contexts that it follows are closed.
func modules(cfg Config, logger *slog.Logger) ([]mcpserver.Module, error) {
	builders := []builder{
		contextOf("rates", rates.New, cfg.Rates, logger),
		contextOf("fuel", fuel.New, cfg.Fuel, logger),
		contextOf("transit", transit.New, cfg.Transit, logger),
		contextOf("offers", offers.New, cfg.Offers, logger),
		contextOf("postal", postal.New, cfg.Postal, logger),
		contextOf("cinema", cinema.New, cfg.Cinema, logger),
		contextOf("stats", stats.New, cfg.Stats, logger),
	}

	built := make([]mcpserver.Module, 0, len(builders))

	for _, next := range builders {
		module, err := next.build()
		if err != nil {
			for _, done := range built {
				if closer, ok := done.(io.Closer); ok {
					_ = closer.Close()
				}
			}

			return nil, fmt.Errorf("building the %s context: %w", next.name, err)
		}

		built = append(built, module)
	}

	return built, nil
}
