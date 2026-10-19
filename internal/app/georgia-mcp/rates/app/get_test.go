package app_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const usd domain.CurrencyCode = "USD"

// NBGRATE-SC-005: With no date, the use case asks for the Tbilisi date of the request.
func TestGetUsesTheTbilisiDate(t *testing.T) {
	t.Parallel()

	source := &fakeSource{
		first:  mustDate(t, "2026-01-01"),
		newest: mustDate(t, "2026-10-18"),
		codes:  []domain.CurrencyCode{usd},
	}
	getRates := app.NewGetOfficialRates(source, clock("2026-10-17T21:30:00Z"))

	if _, err := getRates.Get(
		t.Context(),
		app.OfficialRatesQuery{Language: kernel.LanguageEnglish},
	); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got := source.dates(); len(got) != 1 || got[0].String() != "2026-10-18" {
		t.Errorf("the source received %v, want [2026-10-18]", got)
	}
}

// NBGRATE-SC-007: A date after today with an older set is not published. A published
// set of tomorrow is returned.
func TestGetRefusesADateAfterTheNewestSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		now        string
		requested  string
		newest     string
		wantNewest string
	}{
		{now: "2026-10-16T14:00:00Z", requested: newestDate, newest: newestDate},
		{
			now:        "2026-10-16T10:00:00Z",
			requested:  newestDate,
			newest:     "2026-10-16",
			wantNewest: "2026-10-16",
		},
		{
			now:        "2026-10-18T10:00:00Z",
			requested:  "2026-10-25",
			newest:     newestDate,
			wantNewest: newestDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.now+" "+tt.requested, func(t *testing.T) {
			t.Parallel()

			source := &fakeSource{
				first:  mustDate(t, "2026-01-01"),
				newest: mustDate(t, tt.newest),
				codes:  []domain.CurrencyCode{usd},
			}
			requested := mustDate(t, tt.requested)

			got, err := app.NewGetOfficialRates(source, clock(tt.now)).
				Get(t.Context(), app.OfficialRatesQuery{Date: &requested, Language: kernel.LanguageEnglish})

			assertPublished(t, got, err, requested, tt.wantNewest)
		})
	}
}

func assertPublished(
	t *testing.T,
	got app.OfficialRates,
	err error,
	requested kernel.Date,
	wantNewest string,
) {
	t.Helper()

	if wantNewest == "" {
		if err != nil || got.Set.ValidFrom != requested {
			t.Fatalf("Get() = %v, %v, want the set of %s", got.Set.ValidFrom, err, requested)
		}

		return
	}

	notPublished, ok := errors.AsType[*app.NotPublishedError](err)
	if !ok || !errors.Is(err, app.ErrNotPublished) {
		t.Fatalf("Get() error = %v, want a NotPublishedError", err)
	}

	if notPublished.Newest.String() != wantNewest || notPublished.Requested != requested {
		t.Errorf("NotPublishedError = %+v, want newest %s", notPublished, wantNewest)
	}
}

// NBGRATE-SC-010: A requested code with no rate is missing. The other rates remain.
func TestGetListsTheMissingCodes(t *testing.T) {
	t.Parallel()

	source := &fakeSource{
		first:  mustDate(t, "2026-01-01"),
		newest: mustDate(t, newestDate),
		codes:  []domain.CurrencyCode{"EUR", usd},
	}
	requested := mustDate(t, newestDate)

	got, err := app.NewGetOfficialRates(source, clock("2026-10-18T10:00:00Z")).
		Get(t.Context(), app.OfficialRatesQuery{
			Currencies: []domain.CurrencyCode{usd, "XXX"},
			Date:       &requested,
			Language:   kernel.LanguageEnglish,
		})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(got.Set.Rates) != 1 || got.Set.Rates[0].Currency != usd {
		t.Errorf("rates = %v, want USD only", got.Set.Rates)
	}

	if diff := cmp.Diff([]domain.CurrencyCode{"XXX"}, got.Missing); diff != "" {
		t.Errorf("missing mismatch (-want +got):\n%s", diff)
	}
}

// NBGRATE-SC-028: A set valid after the requested date breaks the invariant, so the
// use case returns no set.
func TestGetRefusesASetAfterTheRequestedDate(t *testing.T) {
	t.Parallel()

	requested := mustDate(t, "2026-10-10")
	source := &fakeSource{override: func(kernel.Date) (domain.RateSet, bool) {
		return setOf(mustDate(t, "2026-10-12"), usd), true
	}}

	got, err := app.NewGetOfficialRates(source, clock("2026-10-18T10:00:00Z")).
		Get(t.Context(), app.OfficialRatesQuery{Date: &requested, Language: kernel.LanguageEnglish})
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Fatalf("Get() error = %v, want ErrSourceUnavailable", err)
	}

	if !got.Set.Empty() {
		t.Errorf("Get() returned a set: %v", got.Set)
	}
}
