package cinema

import (
	"log/slog"
	"time"
)

// NewWithClock builds the module with a fixed clock, for a test.
func NewWithClock(cfg Config, logger *slog.Logger, now func() time.Time) (*Module, error) {
	return newModule(cfg, logger, now)
}
