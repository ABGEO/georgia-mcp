package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// fakeSource answers each match with its applications and its total, and counts the reads.
type fakeSource struct {
	applications []domain.Application
	total        int
	matches      int
}

func (f *fakeSource) Match(context.Context, app.SearchFilter) (app.MatchRead, error) {
	f.matches++

	return app.MatchRead{Applications: f.applications, Total: f.total, ReadAt: clock()}, nil
}

func (f *fakeSource) Application(
	_ context.Context,
	number domain.ApplicationNumber,
) (app.ApplicationRead, error) {
	return app.ApplicationRead{
		Application: domain.Application{Number: number},
		ReadAt:      clock(),
	}, nil
}

func application(t *testing.T, number, registered string) domain.Application {
	t.Helper()

	parsed, err := domain.ParseApplicationNumber(number)
	if err != nil {
		t.Fatal(err)
	}

	date, err := kernel.ParseDate(registered)
	if err != nil {
		t.Fatal(err)
	}

	return domain.Application{Number: parsed, Registered: date}
}

func codeFilter(t *testing.T) app.SearchFilter {
	t.Helper()

	code, err := domain.ParseCadastralCode("01.18.04.020.005")
	if err != nil {
		t.Fatal(err)
	}

	return app.SearchFilter{Code: code}
}

// TASPERM-SC-012: A search sorts the applications newest first, takes the limit, and
// keeps the total.
func TestSearchSortsAndCuts(t *testing.T) {
	t.Parallel()

	source := &fakeSource{
		applications: []domain.Application{
			application(t, "AR11013808", "2024-02-02"),
			application(t, "AR11158278", "2026-08-26"),
			application(t, "AR11027996", "2024-05-14"),
			application(t, "AR1506582", "2017-05-24"),
			application(t, "AR11165994", "2026-10-20"),
		},
		total: 5,
	}

	list, err := app.NewSearchApplications(source).Search(t.Context(), app.SearchQuery{
		Filter: codeFilter(t),
		Limit:  3,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	got := make([]string, 0, len(list.Applications))
	for _, item := range list.Applications {
		got = append(got, item.Number.String())
	}

	want := []string{"AR11165994", "AR11158278", "AR11027996"}
	if !slices.Equal(got, want) || list.Total != 5 || !list.ReadAt.Equal(clock()) {
		t.Errorf("Search() = %v, total %d, want %v, total 5", got, list.Total, want)
	}
}

// TASPERM-SC-013: A search with no filter reads nothing.
func TestSearchNeedsAFilter(t *testing.T) {
	t.Parallel()

	source := &fakeSource{}

	_, err := app.NewSearchApplications(source).Search(t.Context(), app.SearchQuery{Limit: 10})
	if !errors.Is(err, app.ErrNoFilter) || source.matches != 0 {
		t.Errorf("Search() error = %v, reads %d, want ErrNoFilter and no read", err, source.matches)
	}
}

// TASPERM-SC-013: A total above 500 fails with the count. A total of 500 succeeds.
func TestSearchRefusesTooManyMatches(t *testing.T) {
	t.Parallel()

	search := app.NewSearchApplications(&fakeSource{total: 2128})

	_, err := search.Search(t.Context(), app.SearchQuery{Filter: codeFilter(t), Limit: 10})

	var tooMany *app.TooManyMatchesError
	if !errors.As(err, &tooMany) || tooMany.Count != 2128 ||
		!errors.Is(err, app.ErrTooManyMatches) {
		t.Errorf("Search() error = %v, want a TooManyMatchesError of 2128", err)
	}

	search = app.NewSearchApplications(&fakeSource{total: domain.MaxMatches})
	if _, err := search.Search(
		t.Context(),
		app.SearchQuery{Filter: codeFilter(t), Limit: 10},
	); err != nil {
		t.Errorf("Search() with 500 error = %v, want none", err)
	}
}

// TASPERM-SC-015: A number gives its application.
func TestGetApplication(t *testing.T) {
	t.Parallel()

	number, err := domain.ParseApplicationNumber("AR11168188")
	if err != nil {
		t.Fatal(err)
	}

	read, err := app.NewGetApplication(&fakeSource{}).Get(t.Context(), number)
	if err != nil || read.Application.Number != number {
		t.Errorf("Get() = %+v, %v, want AR11168188", read, err)
	}
}
