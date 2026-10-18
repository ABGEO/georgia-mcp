package app_test

import (
	"errors"
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func historySource(t *testing.T) *fakeSource {
	t.Helper()

	return &fakeSource{
		first:  mustDate(t, "2026-09-01"),
		newest: mustDate(t, newestDate),
		codes:  []domain.CurrencyCode{usd},
	}
}

func listHistory(
	t *testing.T,
	source app.OfficialRateSource,
	from, to string,
) (app.OfficialHistory, error) {
	t.Helper()

	return app.NewListOfficialHistory(source, clock("2026-10-18T10:00:00Z")).
		List(t.Context(), app.OfficialHistoryQuery{
			Currency: usd,
			From:     mustDate(t, from),
			To:       mustDate(t, to),
			Language: kernel.LanguageEnglish,
		})
}

// NBGRATE-SC-016: The history starts with the set in force on the first day, and holds
// each valid date once, ascending.
func TestListStartsWithTheSetInForce(t *testing.T) {
	t.Parallel()

	got, err := listHistory(t, historySource(t), "2026-10-04", newestDate)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got.Points) == 0 || got.Points[0].ValidFrom.String() != "2026-10-03" {
		t.Fatalf("first point = %v, want the set valid from 2026-10-03", got.Points)
	}

	seen := map[domain.Date]bool{}

	for i, point := range got.Points {
		if seen[point.ValidFrom] {
			t.Errorf("valid date %s appears twice", point.ValidFrom)
		}

		seen[point.ValidFrom] = true

		if i > 0 && !point.ValidFrom.After(got.Points[i-1].ValidFrom) {
			t.Errorf("points do not ascend at %d", i)
		}
	}
}

// NBGRATE-SC-017: The period has at most 31 days, an end after its start, and no day
// after the newest set.
func TestListValidatesThePeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from, to   string
		wantPoints int
		wantErr    error
	}{
		{from: "2026-10-15", to: "2026-10-15", wantPoints: 1},
		{from: "2026-09-17", to: newestDate, wantPoints: 27},
		{from: "2026-09-16", to: newestDate, wantErr: app.ErrPeriodTooLong},
		{from: "2026-10-10", to: "2026-10-01", wantErr: app.ErrPeriodReversed},
		{from: "2026-10-10", to: "2026-10-25", wantErr: app.ErrNotPublished},
	}

	for _, tt := range tests {
		t.Run(tt.from+" "+tt.to, func(t *testing.T) {
			t.Parallel()

			got, err := listHistory(t, historySource(t), tt.from, tt.to)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("List() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && len(got.Points) != tt.wantPoints {
				t.Errorf("List() = %d points, want %d", len(got.Points), tt.wantPoints)
			}
		})
	}
}

func TestListReportsTheLengthOfALongPeriod(t *testing.T) {
	t.Parallel()

	_, err := listHistory(t, historySource(t), "2026-09-08", newestDate)

	tooLong, ok := errors.AsType[*app.PeriodTooLongError](err)
	if !ok || tooLong.Days != 40 || tooLong.Max != app.MaxHistoryDays {
		t.Errorf("List() error = %v, want a PeriodTooLongError of 40 days", err)
	}
}
