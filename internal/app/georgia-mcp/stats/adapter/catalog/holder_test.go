package catalog_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/catalog"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

var errDown = errors.New("geostat: down")

type call struct {
	Language kernel.Language
	At       time.Duration
}

// fakeSource takes a time for each read, and fails the reads that fail says.
type fakeSource struct {
	start time.Time
	takes time.Duration
	fail  func(number int) bool

	mu    sync.Mutex
	calls []call
}

func (f *fakeSource) Catalog(ctx context.Context, language kernel.Language) (app.Catalog, error) {
	f.mu.Lock()
	f.calls = append(f.calls, call{Language: language, At: time.Since(f.start)})
	number := len(f.calls)
	f.mu.Unlock()

	select {
	case <-ctx.Done():
		return app.Catalog{}, fmt.Errorf("reading: %w", ctx.Err())
	case <-time.After(f.takes):
	}

	if f.fail != nil && f.fail(number) {
		return app.Catalog{}, errDown
	}

	return app.Catalog{ReadAt: time.Now()}, nil
}

func (f *fakeSource) log() []call {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]call{}, f.calls...)
}

func start(t *testing.T, source *fakeSource) *catalog.Holder {
	t.Helper()

	source.start = time.Now()
	holder := catalog.NewHolder(source, slog.New(slog.DiscardHandler))
	holder.Start()
	t.Cleanup(func() { _ = holder.Close() })

	return holder
}

// at sleeps until the instant after the start, then lets the job run.
func at(t *testing.T, begin time.Time, after time.Duration) {
	t.Helper()

	time.Sleep(time.Until(begin.Add(after)))
	synctest.Wait()
}

// GEOSTAT-SC-018: The holder reads en at once, then ka. Before the first read of a
// language, Catalog gives ErrCatalogLoading.
func TestHolderReadsAtStart(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &fakeSource{takes: 5 * time.Minute}
		holder := start(t, source)
		begin := source.start

		at(t, begin, 4*time.Minute)

		if _, err := holder.Catalog(
			t.Context(),
			kernel.LanguageEnglish,
		); !errors.Is(
			err,
			app.ErrCatalogLoading,
		) {
			t.Errorf("Catalog(en) at 4 minutes: error = %v, want ErrCatalogLoading", err)
		}

		at(t, begin, 5*time.Minute)

		if _, err := holder.Catalog(t.Context(), kernel.LanguageEnglish); err != nil {
			t.Errorf("Catalog(en) at 5 minutes: error = %v", err)
		}

		if _, err := holder.Catalog(
			t.Context(),
			kernel.LanguageGeorgian,
		); !errors.Is(
			err,
			app.ErrCatalogLoading,
		) {
			t.Errorf("Catalog(ka) at 5 minutes: error = %v, want ErrCatalogLoading", err)
		}

		at(t, begin, 10*time.Minute)

		if _, err := holder.Catalog(t.Context(), kernel.LanguageGeorgian); err != nil {
			t.Errorf("Catalog(ka) at 10 minutes: error = %v", err)
		}

		want := []call{
			{kernel.LanguageEnglish, 0},
			{kernel.LanguageGeorgian, 5 * time.Minute},
		}
		if diff := cmp.Diff(want, source.log()); diff != "" {
			t.Errorf("calls mismatch (-want +got):\n%s", diff)
		}
	})
}

// GEOSTAT-SC-019: A language is read again 24 hours after a read, not before.
func TestHolderReadsAgainAfterADay(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &fakeSource{}
		start(t, source)
		begin := source.start

		at(t, begin, 24*time.Hour-time.Second)

		if got := len(source.log()); got != 2 {
			t.Errorf("calls before 24 hours = %d, want 2", got)
		}

		at(t, begin, 24*time.Hour)

		want := []call{
			{kernel.LanguageEnglish, 0},
			{kernel.LanguageGeorgian, 0},
			{kernel.LanguageEnglish, 24 * time.Hour},
			{kernel.LanguageGeorgian, 24 * time.Hour},
		}
		if diff := cmp.Diff(want, source.log()); diff != "" {
			t.Errorf("calls mismatch (-want +got):\n%s", diff)
		}
	})
}

// GEOSTAT-SC-020: After the reads fail, the catalog is stale from 24 hours, and an error
// from 7 days.
func TestHolderKeepsTheLastCatalog(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &fakeSource{fail: func(number int) bool { return number > 2 }}
		holder := start(t, source)
		begin := source.start

		at(t, begin, 25*time.Hour)

		got, err := holder.Catalog(t.Context(), kernel.LanguageEnglish)
		if err != nil || !got.Stale {
			t.Errorf("Catalog(en) at 25 hours = stale %t, error %v, want stale", got.Stale, err)
		}

		at(t, begin, 7*24*time.Hour)

		_, err = holder.Catalog(t.Context(), kernel.LanguageEnglish)
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("Catalog(en) at 7 days: error = %v, want ErrSourceUnavailable", err)
		}
	})
}

// GEOSTAT-SC-021: A failed read is sent again after 15 minutes. Until then, the language
// is unavailable. A second Start sends no second read. Close stops the job within 10
// seconds, and no read starts after it.
func TestHolderRetriesAndCloses(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		source := &fakeSource{fail: func(number int) bool { return number == 1 }}
		holder := start(t, source)
		holder.Start()

		begin := source.start

		at(t, begin, time.Minute)

		_, err := holder.Catalog(t.Context(), kernel.LanguageEnglish)
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("Catalog(en) after a failed read: error = %v, want ErrSourceUnavailable", err)
		}

		at(t, begin, 15*time.Minute)

		want := []call{
			{kernel.LanguageEnglish, 0},
			{kernel.LanguageGeorgian, 0},
			{kernel.LanguageEnglish, 15 * time.Minute},
		}
		if diff := cmp.Diff(want, source.log()); diff != "" {
			t.Errorf("calls mismatch (-want +got):\n%s", diff)
		}

		closed := time.Now()

		if err := holder.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}

		if took := time.Since(closed); took > 10*time.Second {
			t.Errorf("Close() took %v, want at most 10 seconds", took)
		}

		at(t, begin, 30*24*time.Hour)

		if got := len(source.log()); got != len(want) {
			t.Errorf("calls after Close = %d, want %d", got, len(want))
		}
	})
}
