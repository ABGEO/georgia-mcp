package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// fakeGrid records each call and answers with err, or a read of the slot it gets.
type fakeGrid struct {
	newest int
	at     []time.Time
	err    error
}

func (f *fakeGrid) Newest(context.Context) (app.GridRead, error) {
	f.newest++

	return f.read(time.Time{})
}

func (f *fakeGrid) At(_ context.Context, slot time.Time) (app.GridRead, error) {
	f.at = append(f.at, slot)

	return f.read(slot)
}

func (f *fakeGrid) read(slot time.Time) (app.GridRead, error) {
	if f.err != nil {
		return app.GridRead{}, f.err
	}

	return app.GridRead{State: domain.GridState{Slot: slot}}, nil
}

func tbilisi(hour, minute, second int) time.Time {
	return time.Date(2026, time.November, 16, hour, minute, second, 0, kernel.TbilisiZone())
}

func now() time.Time {
	return tbilisi(12, 1, 30)
}

// GSEGRID-SC-012: A query with no moment reads the newest grid state.
func TestGetGridReadsTheNewest(t *testing.T) {
	t.Parallel()

	source := &fakeGrid{}

	if _, err := app.NewGetGrid(source, now).Get(t.Context(), app.GridQuery{}); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if source.newest != 1 || len(source.at) != 0 {
		t.Errorf("Newest calls = %d, At calls = %v, want 1 and none", source.newest, source.at)
	}
}

// GSEGRID-SC-013: A moment reads the slot that holds it. A moment in the future or more
// than 31 days ago is refused before any call.
func TestGetGridChecksTheMoment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		moment     time.Time
		wantSlot   time.Time
		wantFuture bool
		wantErr    bool
	}{
		{name: "12:01", moment: tbilisi(12, 1, 0), wantSlot: tbilisi(12, 0, 0)},
		{name: "now", moment: now(), wantSlot: tbilisi(12, 0, 0)},
		{name: "future", moment: tbilisi(12, 1, 31), wantErr: true, wantFuture: true},
		{
			name: "31 days and 1 second", moment: now().Add(-domain.Retention - time.Second),
			wantErr: true,
		},
		{
			name: "31 days", moment: now().Add(-domain.Retention),
			wantSlot: domain.SlotOf(now().Add(-domain.Retention)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := &fakeGrid{}

			_, err := app.NewGetGrid(source, now).
				Get(t.Context(), app.GridQuery{Moment: &tt.moment})
			if tt.wantErr {
				checkRefused(t, err, source, tt.wantFuture)

				return
			}

			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if len(source.at) != 1 || !source.at[0].Equal(tt.wantSlot) {
				t.Errorf("At calls = %v, want %v", source.at, tt.wantSlot)
			}
		})
	}
}

func checkRefused(t *testing.T, err error, source *fakeGrid, wantFuture bool) {
	t.Helper()

	var momentErr *app.MomentError
	if !errors.As(err, &momentErr) || momentErr.Future != wantFuture ||
		!errors.Is(err, app.ErrMomentOutOfRange) {
		t.Fatalf("Get() error = %v, want a *MomentError with Future %v", err, wantFuture)
	}

	if !momentErr.Earliest.Equal(now().Add(-domain.Retention)) {
		t.Errorf("Earliest = %v, want now minus 31 days", momentErr.Earliest)
	}

	if source.newest != 0 || len(source.at) != 0 {
		t.Error("a refused moment called the source")
	}
}

// GSEGRID-SC-014: No grid state in the window gives a *NoGridError with the slot where
// the search started.
func TestGetGridReportsNoGrid(t *testing.T) {
	t.Parallel()

	moment := tbilisi(10, 5, 0)

	tests := []struct {
		name  string
		query app.GridQuery
		want  time.Time
	}{
		{name: "newest", query: app.GridQuery{}, want: tbilisi(12, 0, 0)},
		{name: "moment", query: app.GridQuery{Moment: &moment}, want: tbilisi(10, 3, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := &fakeGrid{err: app.ErrNoGrid}

			_, err := app.NewGetGrid(source, now).Get(t.Context(), tt.query)

			var noGrid *app.NoGridError
			if !errors.As(err, &noGrid) || !errors.Is(err, app.ErrNoGrid) ||
				!noGrid.Slot.Equal(tt.want) {
				t.Errorf("Get() error = %v, want a *NoGridError at %v", err, tt.want)
			}
		})
	}
}
