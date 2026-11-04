// Package catalog holds the catalog of Geostat in each language, and reads it again on a
// clock.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	refreshAfter = 24 * time.Hour
	retryAfter   = 15 * time.Minute
	staleFrom    = 24 * time.Hour
	expiredFrom  = 7 * 24 * time.Hour
	closeWithin  = 10 * time.Second
)

// ErrCloseTimeout tells that the read of the catalog did not stop in time.
var ErrCloseTimeout = errors.New("catalog: the read did not stop within 10 seconds")

// Holder implements app.CatalogSource with a catalog that a background job reads.
type Holder struct {
	source app.CatalogSource
	logger *slog.Logger

	mu   sync.RWMutex
	held map[kernel.Language]app.Catalog
	// failed marks a language whose last read failed.
	failed map[kernel.Language]bool

	cancel context.CancelFunc
	done   chan struct{}
}

var _ app.CatalogSource = (*Holder)(nil)

// NewHolder returns a holder over the source. It reads nothing before Start.
func NewHolder(source app.CatalogSource, logger *slog.Logger) *Holder {
	return &Holder{
		source: source,
		logger: logger,
		held:   map[kernel.Language]app.Catalog{},
		failed: map[kernel.Language]bool{},
	}
}

// Start runs the job: it reads English, then Georgian, at once, then each language again
// on its clock. A second Start does nothing.
func (h *Holder) Start() {
	if h.done != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.done = make(chan struct{})

	go h.run(ctx)
}

// Close stops the job and waits for it at most 10 seconds.
func (h *Holder) Close() error {
	if h.cancel == nil {
		return nil
	}

	h.cancel()

	timer := time.NewTimer(closeWithin)
	defer timer.Stop()

	select {
	case <-h.done:
		return nil
	case <-timer.C:
		return ErrCloseTimeout
	}
}

// Catalog returns the held catalog of the language. It is stale from 24 hours, and
// unavailable from 7 days.
func (h *Holder) Catalog(_ context.Context, language kernel.Language) (app.Catalog, error) {
	h.mu.RLock()
	held, found := h.held[language]
	failed := h.failed[language]
	h.mu.RUnlock()

	if !found && failed {
		return app.Catalog{}, fmt.Errorf("the last read of the catalog of %s failed: %w",
			language, kernel.ErrSourceUnavailable)
	}

	if !found {
		return app.Catalog{}, app.ErrCatalogLoading
	}

	age := time.Since(held.ReadAt)
	if age >= expiredFrom {
		return app.Catalog{}, fmt.Errorf("the catalog of %s is %v old: %w",
			language, age.Round(time.Hour), kernel.ErrSourceUnavailable)
	}

	held.Stale = age >= staleFrom

	return held, nil
}

func (h *Holder) run(ctx context.Context) {
	defer close(h.done)

	now := time.Now()
	schedule := []struct {
		language kernel.Language
		next     time.Time
	}{
		{kernel.LanguageEnglish, now},
		{kernel.LanguageGeorgian, now},
	}

	for {
		due := &schedule[0]
		for i := range schedule {
			if schedule[i].next.Before(due.next) {
				due = &schedule[i]
			}
		}

		if !wait(ctx, time.Until(due.next)) {
			return
		}

		due.next = time.Now().Add(h.read(ctx, due.language))
	}
}

// read reads one language and returns the time until its next read.
func (h *Holder) read(ctx context.Context, language kernel.Language) time.Duration {
	catalog, err := h.source.Catalog(ctx, language)
	if err != nil {
		if ctx.Err() != nil {
			return retryAfter
		}

		h.logger.WarnContext(ctx, "reading the catalog of Geostat",
			slog.String("language", string(language)), slog.Any("error", err))

		h.mu.Lock()
		h.failed[language] = true
		h.mu.Unlock()

		return retryAfter
	}

	h.mu.Lock()
	h.held[language] = catalog
	h.failed[language] = false
	h.mu.Unlock()

	return refreshAfter
}

// wait sleeps for the duration and reports false when the context ends first.
func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
