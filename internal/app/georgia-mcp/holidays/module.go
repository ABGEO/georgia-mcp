package holidays

import (
	"fmt"
	"log/slog"
	"time"

	holidaycache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/adapter/cache"
	nageradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/adapter/nager"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	nagerclient "github.com/abgeo/georgia-mcp/pkg/nager"
)

const switchVariable = "GEOMCP_HOLIDAYS_NAGER_ENABLED"

// Module holds the use cases of the context and their cache. A module of a Nager.Date
// that is off holds none.
type Module struct {
	uses      *tool.UseCases
	calendars *holidaycache.CalendarCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the cache, and the use cases, unless the operator
// turned Nager.Date off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "holidays"))

	if !cfg.Nager.Enabled {
		logger.Info("Nager.Date is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	adapter := nageradapter.NewAdapter(
		nagerclient.New(
			cfg.Nager.ClientOptions(nil, logger.With(slog.String("vendor", "Nager.Date")))...),
		now,
	)

	calendars, err := holidaycache.NewCalendarCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the holidays: %w", err)
	}

	logger.Debug("built the context")

	return &Module{
		uses: &tool.UseCases{
			ListHolidays: app.NewListHolidays(calendars, now),
			GetDay:       app.NewGetDay(calendars, now),
		},
		calendars: calendars,
	}, nil
}

// Register adds the tools of the context. A Nager.Date that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the cache.
func (m *Module) Close() error {
	if m.uses == nil {
		return nil
	}

	return m.calendars.Close()
}
