// Package transit is the bounded context of the public transport of Tbilisi: the stops,
// the routes, the arrivals, the timetables, and the trip plans of the TTC.
package transit

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	transitcache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/adapter/tool"
	ttcadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/adapter/ttc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/transit/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	ttcclient "github.com/abgeo/georgia-mcp/pkg/ttc"
)

const switchVariable = "GEOMCP_TRANSIT_TTC_ENABLED"

// Module holds the use cases of the context and its caches. A module of a TTC that is
// off holds none.
type Module struct {
	uses    *tool.UseCases
	closers []func() error
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the caches, and the use cases, unless the
// operator turned the TTC off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "transit"))

	if !cfg.TTC.Enabled {
		logger.Info("TTC is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	client := ttcclient.New(
		ttcclient.WithBaseURL(cfg.TTC.BaseURL.String()),
		ttcclient.WithHTTPClient(httpclient.New(httpclient.Config(cfg.TTC.HTTP))),
		ttcclient.WithAPIKey(cfg.TTC.APIKey),
	)
	adapter := ttcadapter.NewAdapter(client, now, logger)

	module := &Module{}

	uses, err := module.build(adapter, now)
	if err != nil {
		_ = module.Close()

		return nil, err
	}

	module.uses = uses

	logger.Debug("built the context")

	return module, nil
}

// Register adds the tools of the context. A TTC that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops every cache.
func (m *Module) Close() error {
	errs := make([]error, 0, len(m.closers))
	for _, closeCache := range m.closers {
		errs = append(errs, closeCache())
	}

	return errors.Join(errs...)
}

func (m *Module) build(adapter *ttcadapter.Adapter, now func() time.Time) (*tool.UseCases, error) {
	network, err := transitcache.NewNetworkCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the network: %w", err)
	}

	m.closers = append(m.closers, network.Close)

	routes, err := transitcache.NewRouteCache(adapter, now)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the routes: %w", err)
	}

	m.closers = append(m.closers, routes.Close)

	arrivals, err := transitcache.NewArrivalCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the arrivals: %w", err)
	}

	m.closers = append(m.closers, arrivals.Close)

	places, err := transitcache.NewPlaceCache(adapter)
	if err != nil {
		return nil, fmt.Errorf("building the cache of the places: %w", err)
	}

	m.closers = append(m.closers, places.Close)

	return &tool.UseCases{
		SearchStops:  app.NewSearchStops(network),
		NearbyStops:  app.NewListNearbyStops(network),
		Arrivals:     app.NewListArrivals(network, arrivals),
		SearchRoutes: app.NewSearchRoutes(network),
		Route:        app.NewGetRoute(network, routes),
		Timetable:    app.NewGetTimetable(network, routes, now),
		Trip:         app.NewPlanTrip(network, adapter, now),
		Places:       app.NewSearchPlaces(places),
	}, nil
}
