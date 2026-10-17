package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/httprate"
)

const (
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 120 * time.Second
	maxHeaderBytes    = 64 << 10
	shutdownTimeout   = 10 * time.Second
	unknownClient     = "unknown"
)

// Server serves the MCP handler and the probes.
type Server struct {
	cfg       Config
	router    http.Handler
	http      *http.Server
	ready     atomic.Bool
	listening chan struct{}
	addr      net.Addr
}

// New returns a server that is not ready. Run starts it.
func New(cfg Config, logger *slog.Logger, mcpHandler http.Handler) *Server {
	srv := &Server{cfg: cfg, listening: make(chan struct{})}
	srv.router = srv.routes(logger, mcpHandler)
	srv.http = &http.Server{
		Handler:           srv.router,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	return srv
}

// Handler returns the router with its middleware.
func (s *Server) Handler() http.Handler {
	return s.router
}

// SetReady sets the answer of the readiness probe.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// Addr returns the bound address. It waits until Run listens.
func (s *Server) Addr() net.Addr {
	<-s.listening

	return s.addr
}

// Run serves until the context is cancelled. Then it reports that it is not ready
// and lets each request in progress end, within 10 seconds.
func (s *Server) Run(ctx context.Context) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.cfg.Address)
	if err != nil {
		return fmt.Errorf("serving http: %w", err)
	}

	s.addr = listener.Addr()
	close(s.listening)

	served := make(chan error, 1)

	go func() { served <- s.http.Serve(listener) }()

	select {
	case err := <-served:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("serving http: %w", err)
	case <-ctx.Done():
	}

	s.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down http: %w", err)
	}

	return nil
}

func (s *Server) routes(logger *slog.Logger, mcpHandler http.Handler) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)

	if len(s.cfg.TrustedProxies) > 0 {
		router.Use(middleware.ClientIPFromXFF(s.cfg.TrustedProxies...))
	} else {
		router.Use(middleware.ClientIPFromRemoteAddr)
	}

	router.Use(httplog.RequestLogger(
		logger.With(slog.String("component", "middleware"), slog.String("middleware", "access")),
		&httplog.Options{
			Level:         slog.LevelDebug,
			Schema:        httplog.SchemaOTEL,
			RecoverPanics: false,
			Skip:          skipSuccessfulProbe,
		},
	))
	router.Use(middleware.Recoverer)

	router.Get("/livez", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "ok")
	})
	router.Get("/readyz", s.readyz)

	router.Group(func(group chi.Router) {
		group.Use(allowOrigins(s.cfg.AllowedOrigins))
		group.Use(httprate.LimitBy(s.cfg.RateLimit, s.cfg.RateWindow, clientAddress))
		group.Handle("/mcp", mcpHandler)
	})

	return router
}

func skipSuccessfulProbe(req *http.Request, status int) bool {
	return status == http.StatusOK && (req.URL.Path == "/livez" || req.URL.Path == "/readyz")
}

func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	if s.ready.Load() {
		writeText(w, http.StatusOK, "ready")

		return
	}

	writeText(w, http.StatusServiceUnavailable, "not ready")
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// clientAddress keys the rate limit. A request whose address the middleware could
// not resolve shares one bucket, so a malformed header buys no fresh limit.
func clientAddress(r *http.Request) (string, error) {
	if address := middleware.GetClientIP(r.Context()); address != "" {
		return address, nil
	}

	return unknownClient, nil
}

func allowOrigins(allowed []string) func(http.Handler) http.Handler {
	origins := make(map[string]struct{}, len(allowed))
	for _, origin := range allowed {
		origins[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" {
				if _, ok := origins[origin]; !ok {
					writeText(w, http.StatusForbidden, "forbidden origin")

					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
