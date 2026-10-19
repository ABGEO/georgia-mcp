package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const newestDate = "2026-10-17"

// fakeSource answers like the NBG: a set for each day except Sunday from first to
// newest, the newest set for a later date, and an empty set before first.
type fakeSource struct {
	first, newest kernel.Date
	codes         []domain.CurrencyCode
	override      func(date kernel.Date) (domain.RateSet, bool)

	mu        sync.Mutex
	requested []kernel.Date
}

func (f *fakeSource) RateSet(
	_ context.Context,
	date kernel.Date,
	_ kernel.Language,
) (app.RateSetResult, error) {
	f.mu.Lock()
	f.requested = append(f.requested, date)
	f.mu.Unlock()

	if f.override != nil {
		if set, ok := f.override(date); ok {
			return app.RateSetResult{Set: set}, nil
		}
	}

	if date.Before(f.first) {
		return app.RateSetResult{}, nil
	}

	valid := date
	if valid.After(f.newest) {
		valid = f.newest
	}

	for weekday(valid) == time.Sunday {
		valid = valid.AddDays(-1)
	}

	return app.RateSetResult{Set: setOf(valid, f.codes...)}, nil
}

func (f *fakeSource) dates() []kernel.Date {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]kernel.Date(nil), f.requested...)
}

func weekday(date kernel.Date) time.Weekday {
	return date.Time().Weekday()
}

func setOf(validFrom kernel.Date, codes ...domain.CurrencyCode) domain.RateSet {
	rates := make([]domain.OfficialRate, 0, len(codes))

	for _, code := range codes {
		rate, _ := domain.NewOfficialRate(
			code,
			string(code),
			1,
			decimal.RequireFromString("2.5982"),
			decimal.Zero,
		)
		rates = append(rates, rate)
	}

	return domain.NewRateSet(validFrom, time.Time{}, rates)
}

func mustDate(t *testing.T, value string) kernel.Date {
	t.Helper()

	date, err := kernel.ParseDate(value)
	if err != nil {
		t.Fatal(err)
	}

	return date
}

func clock(value string) func() time.Time {
	instant, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}

	return func() time.Time { return instant }
}
