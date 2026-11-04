package stats

import (
	"log/slog"
	"net/http"
)

// NewWithTransport builds the module over a transport, for a test.
func NewWithTransport(
	cfg Config,
	logger *slog.Logger,
	transport http.RoundTripper,
) (*Module, error) {
	return newModule(cfg, logger, transport)
}
