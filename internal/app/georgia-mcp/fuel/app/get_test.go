package app_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func gradesOf(prices app.CompanyPrices) []domain.Grade {
	grades := make([]domain.Grade, 0, len(prices.Prices))
	for _, price := range prices.Prices {
		grades = append(grades, price.Grade)
	}

	return grades
}

func codesOf(answer app.Prices) []domain.CompanyCode {
	codes := make([]domain.CompanyCode, 0, len(answer.Companies))
	for _, company := range answer.Companies {
		codes = append(codes, company.Company.Code)
	}

	return codes
}

// WISFUEL-SC-008: No company asks every company, sorted by code. No grade asks every
// grade. A grade keeps the products of that grade only.
func TestGetPricesAsksEveryCompanyAndGrade(t *testing.T) {
	t.Parallel()

	use := app.NewGetPrices([]app.PriceSource{
		newFake(wissol, domain.GradeDiesel, domain.GradeRegular),
		newFake(beta, domain.GradeDiesel, domain.GradeRegular),
	}, slog.New(slog.DiscardHandler))

	every, err := use.Get(t.Context(), app.PricesQuery{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if diff := cmp.Diff([]domain.CompanyCode{beta, wissol}, codesOf(every)); diff != "" {
		t.Errorf("companies (-want +got):\n%s", diff)
	}

	for _, company := range every.Companies {
		want := []domain.Grade{domain.GradeRegular, domain.GradeDiesel}
		if diff := cmp.Diff(want, gradesOf(company)); diff != "" {
			t.Errorf("%s grades, sorted by grade (-want +got):\n%s", company.Company.Code, diff)
		}
	}

	diesel, err := use.Get(t.Context(), app.PricesQuery{Grades: []domain.Grade{domain.GradeDiesel}})
	if err != nil {
		t.Fatalf("Get(diesel) error = %v", err)
	}

	for _, company := range diesel.Companies {
		if diff := cmp.Diff([]domain.Grade{domain.GradeDiesel}, gradesOf(company)); diff != "" {
			t.Errorf("%s grades (-want +got):\n%s", company.Company.Code, diff)
		}
	}
}

// WISFUEL-SC-009: An unknown company is an error that names the known codes, and no
// company is asked.
func TestGetPricesRefusesAnUnknownCompany(t *testing.T) {
	t.Parallel()

	source := newFake(wissol, domain.GradeRegular)
	use := app.NewGetPrices([]app.PriceSource{source}, slog.New(slog.DiscardHandler))

	_, err := use.Get(t.Context(), app.PricesQuery{Companies: []domain.CompanyCode{"gulf"}})

	unknown, ok := errors.AsType[*app.UnknownCompanyError](err)
	if !ok {
		t.Fatalf("Get() error = %v, want an *UnknownCompanyError", err)
	}

	want := app.UnknownCompanyError{Code: "gulf", Known: []domain.CompanyCode{wissol}}
	if diff := cmp.Diff(want, *unknown); diff != "" {
		t.Errorf("UnknownCompanyError (-want +got):\n%s", diff)
	}

	if !errors.Is(err, app.ErrUnknownCompany) {
		t.Errorf("errors.Is(%v, ErrUnknownCompany) = false", err)
	}

	if calls := source.calls.Load(); calls != 0 {
		t.Errorf("the company was asked %d times, want 0", calls)
	}
}

// WISFUEL-SC-010: A grade named twice counts once. A grade that the company does not
// sell is missing.
func TestGetPricesNamesTheMissingGrades(t *testing.T) {
	t.Parallel()

	use := app.NewGetPrices(
		[]app.PriceSource{newFake(wissol, domain.GradeRegular)},
		slog.New(slog.DiscardHandler),
	)

	answer, err := use.Get(t.Context(), app.PricesQuery{
		Grades:    []domain.Grade{domain.GradeCNG, domain.GradeRegular, domain.GradeRegular},
		Companies: []domain.CompanyCode{wissol, wissol},
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(answer.Companies) != 1 {
		t.Fatalf("companies = %v, want wissol once", codesOf(answer))
	}

	company := answer.Companies[0]
	if diff := cmp.Diff([]domain.Grade{domain.GradeRegular}, gradesOf(company)); diff != "" {
		t.Errorf("grades (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]domain.Grade{domain.GradeCNG}, company.Missing); diff != "" {
		t.Errorf("missing (-want +got):\n%s", diff)
	}
}

type failureCase struct {
	name            string
	wissol          *fakeSource
	companies       []domain.CompanyCode
	wantCompanies   []domain.CompanyCode
	wantUnavailable []domain.CompanyCode
	wantErr         error
}

func failureCases() []failureCase {
	failure := fmt.Errorf("reading: %w", kernel.ErrSourceUnavailable)

	return []failureCase{
		{
			name:            "one fails",
			wissol:          &fakeSource{err: failure},
			wantCompanies:   []domain.CompanyCode{beta},
			wantUnavailable: []domain.CompanyCode{wissol},
		},
		{
			name:      "the only request fails",
			wissol:    &fakeSource{err: failure},
			companies: []domain.CompanyCode{wissol},
			wantErr:   kernel.ErrSourceUnavailable,
		},
		{
			name: "one waits",
			wissol: &fakeSource{
				wait:   time.Second,
				grades: []domain.Grade{domain.GradeRegular},
			},
			wantCompanies: []domain.CompanyCode{beta, wissol},
		},
	}
}

// WISFUEL-SC-011: A company that fails is unavailable. When every company fails, Get
// fails. A slow company does not delay the other one.
func TestGetPricesReportsAFailedCompany(t *testing.T) {
	t.Parallel()

	for _, tt := range failureCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.wissol.company = domain.Company{Code: wissol, Name: "Wissol"}
			use := app.NewGetPrices(
				[]app.PriceSource{tt.wissol, newFake(beta, domain.GradeRegular)},
				slog.New(slog.DiscardHandler),
			)

			start := time.Now()
			answer, err := use.Get(t.Context(), app.PricesQuery{Companies: tt.companies})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}

			if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
				t.Errorf("Get() took %v, want less than 1.5 seconds", elapsed)
			}

			if tt.wantErr != nil {
				return
			}

			if diff := cmp.Diff(tt.wantCompanies, codesOf(answer)); diff != "" {
				t.Errorf("companies (-want +got):\n%s", diff)
			}

			got := unavailableCodes(answer)
			if diff := cmp.Diff(tt.wantUnavailable, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("unavailable (-want +got):\n%s", diff)
			}
		})
	}
}

func unavailableCodes(answer app.Prices) []domain.CompanyCode {
	codes := make([]domain.CompanyCode, 0, len(answer.Unavailable))
	for _, company := range answer.Unavailable {
		codes = append(codes, company.Code)
	}

	return codes
}

// leftOut returns the vendor and the error of each "source unavailable" record in buf.
func leftOut(t *testing.T, buf *bytes.Buffer) [][2]string {
	t.Helper()

	var got [][2]string

	for line := range strings.Lines(buf.String()) {
		var record struct {
			Level   string `json:"level"`
			Message string `json:"msg"`
			Vendor  string `json:"vendor"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("a log line is not JSON: %q", line)
		}

		if record.Message == "source unavailable" && record.Level == "WARN" {
			got = append(got, [2]string{record.Vendor, record.Error})
		}
	}

	return got
}

func bufferLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer

	return slog.New(slog.NewJSONHandler(&buf, nil)), &buf
}

// PLATFORM-SC-041: A company that the answer leaves out writes one warning with its
// error. When every company fails, the use case writes none.
func TestGetPricesLogsALeftOutCompany(t *testing.T) {
	t.Parallel()

	failure := fmt.Errorf("reading: %w", kernel.ErrSourceUnavailable)

	failing := func(code domain.CompanyCode) *fakeSource {
		source := newFake(code)
		source.err = failure

		return source
	}

	tests := []struct {
		name    string
		sources []app.PriceSource
		want    [][2]string
	}{
		{
			name:    "one fails",
			sources: []app.PriceSource{newFake(wissol, domain.GradeRegular), failing(beta)},
			want:    [][2]string{{"beta", "reading: source: unavailable"}},
		},
		{
			name:    "every one fails",
			sources: []app.PriceSource{failing(wissol), failing(beta)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger, buf := bufferLogger()

			_, _ = app.NewGetPrices(tt.sources, logger).Get(t.Context(), app.PricesQuery{})

			if diff := cmp.Diff(tt.want, leftOut(t, buf)); diff != "" {
				t.Errorf("records (-want +got):\n%s", diff)
			}
		})
	}
}
