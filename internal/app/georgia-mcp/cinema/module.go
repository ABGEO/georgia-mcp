// Package cinema is the bounded context of the cinemas of Cavea: the open cinemas, the
// movies, the sessions with their prices, and the seats of a session.
package cinema

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	cinemacache "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/adapter/cache"
	caveaadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/adapter/cavea"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/cinema/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	caveaclient "github.com/abgeo/georgia-mcp/pkg/cavea"
)

const switchVariable = "GEOMCP_CINEMA_CAVEA_ENABLED"

// Module holds the use cases of the context and its caches. A module of a Cavea that is
// off holds none.
type Module struct {
	uses     *tool.UseCases
	cinemas  *cinemacache.CinemaCache
	movies   *cinemacache.MovieCache
	sessions *cinemacache.SessionCache
}

var _ mcpserver.Module = (*Module)(nil)

// New builds the client, the adapter, the caches, and the use cases, unless the operator
// turned Cavea off.
func New(cfg Config, logger *slog.Logger) (*Module, error) {
	return newModule(cfg, logger, time.Now)
}

func newModule(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	logger = logger.With(slog.String("component", "context"), slog.String("context", "cinema"))

	if !cfg.Cavea.Enabled {
		logger.Info("Cavea is off", slog.String("variable", switchVariable))

		return &Module{}, nil
	}

	adapter := caveaadapter.NewAdapter(caveaclient.New(cfg.Cavea.ClientOptions(nil)...), now)
	module := &Module{}

	var err error

	if module.cinemas, err = cinemacache.NewCinemaCache(adapter); err != nil {
		return nil, fmt.Errorf("building the cache of the cinemas: %w", err)
	}

	if module.movies, err = cinemacache.NewMovieCache(adapter); err != nil {
		_ = module.Close()

		return nil, fmt.Errorf("building the cache of the movies: %w", err)
	}

	if module.sessions, err = cinemacache.NewSessionCache(adapter); err != nil {
		_ = module.Close()

		return nil, fmt.Errorf("building the cache of the sessions: %w", err)
	}

	module.uses = &tool.UseCases{
		ListCinemas:  app.NewListCinemas(module.cinemas),
		ListMovies:   app.NewListMovies(module.movies, now),
		SearchMovies: app.NewSearchMovies(module.movies, now),
		ListSessions: app.NewListSessions(
			module.cinemas,
			module.movies,
			module.sessions,
			adapter,
			now,
			logger,
		),
		GetSeatMap: app.NewGetSeatMap(
			module.cinemas,
			module.movies,
			module.sessions,
			adapter,
			now,
		),
	}

	logger.Debug("built the context")

	return module, nil
}

// Register adds the tools of the context. A Cavea that is off adds none.
func (m *Module) Register(registry *mcpserver.Registry) error {
	if m.uses == nil {
		return nil
	}

	return tool.Register(registry, *m.uses)
}

// Close stops the caches.
func (m *Module) Close() error {
	var errs []error

	if m.cinemas != nil {
		errs = append(errs, m.cinemas.Close())
	}

	if m.movies != nil {
		errs = append(errs, m.movies.Close())
	}

	if m.sessions != nil {
		errs = append(errs, m.sessions.Close())
	}

	return errors.Join(errs...)
}
