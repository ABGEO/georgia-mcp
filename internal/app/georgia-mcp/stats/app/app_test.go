package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const (
	wagesRegion = "Social Statistics/Labour/Wages_Region.px"
	cpiAnnual   = "Price Statistics/CPI/National Consumer Price Index/" +
		"CPI_in_Georgia(same_month_of_the_previous_year=100_orenovani2.px"
	cpiMonthly = "Price Statistics/CPI/National Consumer Price Index/" +
		"CPI_in_Georgia_(previous_month=100)_orenovani2.px"

	region   = "Region"
	year     = "Year"
	tbilisiT = "Tbilisi"
	coreRate = "4.0"
	monthly  = "0.9"
	october  = "2026-10"
	annual   = "5.9"
	average  = "5.2"
)

func code(tb testing.TB, text string) domain.TableCode {
	tb.Helper()

	parsed, err := domain.ParseTableCode(text)
	if err != nil {
		tb.Fatal(err)
	}

	return parsed
}

func figure(tb testing.TB, text string) domain.Figure {
	tb.Helper()

	parsed, err := domain.ParseFigure(text)
	if err != nil {
		tb.Fatal(err)
	}

	return parsed
}

func tbilisi(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 12, 0, 0, 0, kernel.TbilisiZone())
}

type fakeCatalog struct {
	catalog app.Catalog
	err     error
}

func (f fakeCatalog) Catalog(context.Context, kernel.Language) (app.Catalog, error) {
	return f.catalog, f.err
}

func searchCatalog(t *testing.T) app.Catalog {
	t.Helper()

	return app.Catalog{
		Entries: []domain.CatalogEntry{
			{
				Code:    code(t, wagesRegion),
				Title:   "Average Monthly Nominal Salary of Employees by Region,Years and Quarters",
				Folder:  "Social Statistics / Labour",
				Updated: tbilisi(2026, 10, 8),
			},
			{
				Code:    code(t, cpiAnnual),
				Title:   "Consumer Price Indices (Same month of the previous year=100)",
				Folder:  "Price Statistics / CPI / National Consumer Price Index",
				Updated: tbilisi(2026, 10, 1),
			},
			{
				Code:    code(t, cpiMonthly),
				Title:   "Consumer Price Index (Previous month=100)",
				Folder:  "Price Statistics / CPI / National Consumer Price Index",
				Updated: tbilisi(2026, 10, 2),
			},
		},
		ReadAt: tbilisi(2026, 11, 4),
	}
}

func codesOf(entries []domain.CatalogEntry) []string {
	codes := make([]string, 0, len(entries))
	for _, entry := range entries {
		codes = append(codes, entry.Code.String())
	}

	return codes
}

// GEOSTAT-SC-007: A search matches every word in the title, the folder, or the code,
// with no case, and orders the matches from the newest update.
func TestSearchTables(t *testing.T) {
	t.Parallel()

	search := app.NewSearchTables(fakeCatalog{catalog: searchCatalog(t)})

	tests := []struct {
		name      string
		text      string
		limit     int
		want      []string
		wantTotal int
	}{
		{name: "two words", text: "salary region", want: []string{wagesRegion}, wantTotal: 1},
		{
			name: "case", text: "CONSUMER price",
			want: []string{cpiMonthly, cpiAnnual}, wantTotal: 2,
		},
		{name: "no match", text: "zzqx", want: []string{}, wantTotal: 0},
		{name: "limit", text: "consumer", limit: 1, want: []string{cpiMonthly}, wantTotal: 2},
		{name: "folder", text: "labour", want: []string{wagesRegion}, wantTotal: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := search.Search(t.Context(), app.TableQuery{
				Text: tt.text, Language: kernel.LanguageEnglish, Limit: tt.limit,
			})
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, codesOf(got.Entries)); diff != "" {
				t.Errorf("codes mismatch (-want +got):\n%s", diff)
			}

			if got.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", got.Total, tt.wantTotal)
			}
		})
	}
}

// GEOSTAT-SC-007: An empty text and a text of 101 runes are refused.
func TestSearchTablesRefusesABadText(t *testing.T) {
	t.Parallel()

	search := app.NewSearchTables(fakeCatalog{catalog: searchCatalog(t)})

	for _, text := range []string{"", "   ", strings.Repeat("ა", 101)} {
		_, err := search.Search(t.Context(), app.TableQuery{Text: text})
		if !errors.Is(err, app.ErrInvalidQuery) {
			t.Errorf("Search(%q) error = %v, want ErrInvalidQuery", text, err)
		}
	}
}

// recordedCatalog reads the catalog of Geostat in English, as recorded on 2026-11-04.
func recordedCatalog(tb testing.TB) app.Catalog {
	tb.Helper()

	body, err := os.ReadFile("testdata/catalog-en.json")
	if err != nil {
		tb.Fatal(err)
	}

	var rows []struct {
		Code    string `json:"code"`
		Title   string `json:"title"`
		Folder  string `json:"folder"`
		Updated string `json:"updated"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		tb.Fatal(err)
	}

	catalog := app.Catalog{}

	for _, row := range rows {
		updated, err := time.ParseInLocation(
			"2006-01-02T15:04:05",
			row.Updated,
			kernel.TbilisiZone(),
		)
		if err != nil {
			tb.Fatal(err)
		}

		catalog.Entries = append(catalog.Entries, domain.CatalogEntry{
			Code: code(tb, row.Code), Title: row.Title, Folder: row.Folder, Updated: updated,
		})
	}

	return catalog
}

// GEOSTAT-SC-023: A search over the catalog of 1,046 tables takes less than 500 ms.
func TestSearchTablesIsFast(t *testing.T) {
	t.Parallel()

	search := app.NewSearchTables(fakeCatalog{catalog: recordedCatalog(t)})

	start := time.Now()

	if _, err := search.Search(t.Context(), app.TableQuery{Text: "price index"}); err != nil {
		t.Fatal(err)
	}

	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Errorf("Search() took %s, want less than 500 ms", elapsed)
	}
}

// GEOSTAT-SC-023: A search over the catalog of 1,046 tables takes less than 500 ms.
func BenchmarkSearchTables(b *testing.B) {
	catalog := recordedCatalog(b)

	search := app.NewSearchTables(fakeCatalog{catalog: catalog})

	for b.Loop() {
		start := time.Now()

		if _, err := search.Search(b.Context(), app.TableQuery{Text: "price index"}); err != nil {
			b.Fatal(err)
		}

		if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
			b.Fatalf("Search() took %s", elapsed)
		}
	}
}

type fakeTables struct {
	table      domain.Table
	tableErr   error
	figures    map[string]domain.Figure
	selections [][]app.DimensionSelection
}

func (f *fakeTables) Table(
	context.Context,
	domain.TableCode,
	kernel.Language,
) (app.TableDescription, error) {
	return app.TableDescription{Table: f.table, ReadAt: tbilisi(2026, 11, 4)}, f.tableErr
}

func (f *fakeTables) Cells(
	_ context.Context,
	_ domain.TableCode,
	selection []app.DimensionSelection,
) (app.CellSet, error) {
	f.selections = append(f.selections, selection)

	return app.CellSet{Figures: f.figures, ReadAt: tbilisi(2026, 11, 4)}, nil
}

func dimension(code string, texts ...string) domain.Dimension {
	values := make([]domain.DimensionValue, 0, len(texts))
	for i, text := range texts {
		values = append(values, domain.DimensionValue{Code: strconv.Itoa(i), Text: text})
	}

	return domain.Dimension{Code: code, Text: code, Values: values}
}

func years(from, to int) []string {
	texts := []string{}
	for number := from; number <= to; number++ {
		texts = append(texts, strconv.Itoa(number))
	}

	return texts
}

func wagesTable(t *testing.T) domain.Table {
	t.Helper()

	return domain.Table{
		Entry: domain.CatalogEntry{Code: code(t, wagesRegion), Title: "Salaries"},
		Dimensions: []domain.Dimension{
			dimension(region, "Total", tbilisiT, "Adjara A.R.", "Guria", "Imereti", "Kakheti",
				"Mtskheta-Mtianeti", "Racha", "Samegrelo", "Samtskhe-Javakheti", "Kvemo Kartli",
				"Shida Kartli"),
			dimension(year, years(2010, 2026)...),
			dimension("Quarter", "I", "II", "III", "IV", "Annual"),
		},
	}
}

// GEOSTAT-SC-009: A selection resolves the last values and the omitted dimensions, and the
// rows follow the order of the table.
func TestGetCellsResolvesTheSelection(t *testing.T) {
	t.Parallel()

	source := &fakeTables{
		table: wagesTable(t),
		figures: map[string]domain.Figure{
			app.CellKey([]string{"1", "16", "0"}): figure(t, "2510.3"),
			app.CellKey([]string{"1", "16", "1"}): figure(t, "2601.0"),
			app.CellKey([]string{"1", "16", "2"}): figure(t, ".."),
		},
	}

	cells, err := app.NewGetCells(source).Get(t.Context(), app.CellQuery{
		Code: code(t, wagesRegion),
		Selection: []app.DimensionSelection{
			{Dimension: region, Values: []string{"1"}},
			{Dimension: year, Last: 1},
		},
		Language: kernel.LanguageEnglish,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	wantSelection := []app.DimensionSelection{
		{Dimension: region, Values: []string{"1"}},
		{Dimension: year, Values: []string{"16"}},
		{Dimension: "Quarter", Values: []string{"0", "1", "2", "3", "4"}},
	}
	if diff := cmp.Diff(wantSelection, source.selections[0]); diff != "" {
		t.Errorf("selection mismatch (-want +got):\n%s", diff)
	}

	type row struct {
		Values []string
		Figure string
	}

	got := make([]row, 0, len(cells.Rows))
	for _, cell := range cells.Rows {
		got = append(got, row{Values: cell.Values, Figure: cell.Figure.String()})
	}

	want := []row{
		{[]string{tbilisiT, "2026", "I"}, "2510.3"},
		{[]string{tbilisiT, "2026", "II"}, "2601.0"},
		{[]string{tbilisiT, "2026", "III"}, ""},
		{[]string{tbilisiT, "2026", "IV"}, ""},
		{[]string{tbilisiT, "2026", "Annual"}, ""},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rows mismatch (-want +got):\n%s", diff)
	}

	if len(cells.Table.Dimensions[1].Values) != 1 {
		t.Errorf("selected years = %+v, want one", cells.Table.Dimensions[1].Values)
	}
}

// GEOSTAT-SC-009: A count of last values above the length of the dimension gives every
// value.
func TestGetCellsGivesEveryValueForALongLast(t *testing.T) {
	t.Parallel()

	source := &fakeTables{table: wagesTable(t)}

	_, err := app.NewGetCells(source).Get(t.Context(), app.CellQuery{
		Code: code(t, wagesRegion),
		Selection: []app.DimensionSelection{
			{Dimension: region, Values: []string{"0"}},
			{Dimension: year, Last: 50},
		},
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got := len(source.selections[0][1].Values); got != 17 {
		t.Errorf("years = %d, want 17", got)
	}
}

// cpiTable has the dimensions of the annual CPI table: 7 × 13 × 23 × 12 values.
func cpiTable(t *testing.T) domain.Table {
	t.Helper()

	const cpiRegions = "Georgia Tbilisi Kutaisi Batumi Gori Telavi Zugdidi"

	months := strings.Fields("I II III IV V VI VII VIII IX X XI XII")

	return domain.Table{
		Entry: domain.CatalogEntry{Code: code(t, cpiAnnual)},
		Dimensions: []domain.Dimension{
			dimension(region, strings.Fields(cpiRegions)...),
			dimension("Groups", strings.Split("abcdefghijklm", "")...),
			dimension(year, years(2004, 2026)...),
			dimension("month", months...),
		},
	}
}

type badSelection struct {
	name      string
	selection []app.DimensionSelection
	wantErr   error
	want      app.SelectionError
}

func badSelections() []badSelection {
	return []badSelection{
		{
			name: "every cell", wantErr: app.ErrTooManyCells,
			want: app.SelectionError{Problem: app.ProblemTooManyCells, Cells: 25116},
		},
		{
			name:      "unknown dimension",
			selection: []app.DimensionSelection{{Dimension: "Town", Values: []string{"0"}}},
			wantErr:   app.ErrInvalidSelection,
			want: app.SelectionError{
				Problem: app.ProblemUnknownDimension, Dimension: "Town",
				Accepted: []string{region, "Groups", year, "month"},
			},
		},
		{
			name:      "unknown value",
			selection: []app.DimensionSelection{{Dimension: region, Values: []string{"99"}}},
			wantErr:   app.ErrInvalidSelection,
			want: app.SelectionError{
				Problem: app.ProblemUnknownValue, Dimension: region, Value: "99",
				Accepted: []string{"0", "1", "2", "3", "4", "5", "6"},
			},
		},
		{
			name: "values and last",
			selection: []app.DimensionSelection{
				{Dimension: year, Values: []string{"0"}, Last: 1},
			},
			wantErr: app.ErrInvalidSelection,
			want:    app.SelectionError{Problem: app.ProblemValuesOrLast, Dimension: year},
		},
		{
			name:      "neither",
			selection: []app.DimensionSelection{{Dimension: year}},
			wantErr:   app.ErrInvalidSelection,
			want:      app.SelectionError{Problem: app.ProblemValuesOrLast, Dimension: year},
		},
		{
			name:      "a negative last",
			selection: []app.DimensionSelection{{Dimension: year, Last: -1}},
			wantErr:   app.ErrInvalidSelection,
			want:      app.SelectionError{Problem: app.ProblemValuesOrLast, Dimension: year},
		},
		{
			name: "twice",
			selection: []app.DimensionSelection{
				{Dimension: year, Last: 1}, {Dimension: year, Last: 2},
			},
			wantErr: app.ErrInvalidSelection,
			want:    app.SelectionError{Problem: app.ProblemDuplicate, Dimension: year},
		},
	}
}

// GEOSTAT-SC-010: A selection of more than 1,000 cells, or with a bad dimension or value,
// is refused before a call for the cells.
func TestGetCellsRefusesABadSelection(t *testing.T) {
	t.Parallel()

	cpi := cpiTable(t)

	for _, tt := range badSelections() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := &fakeTables{table: cpi}

			_, err := app.NewGetCells(source).Get(t.Context(), app.CellQuery{
				Code: code(t, cpiAnnual), Selection: tt.selection,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}

			var selectionErr *app.SelectionError
			if !errors.As(err, &selectionErr) {
				t.Fatalf("Get() error = %v, want a *SelectionError", err)
			}

			ignore := cmpopts.IgnoreUnexported(app.SelectionError{})
			if diff := cmp.Diff(tt.want, *selectionErr, ignore); diff != "" {
				t.Errorf("SelectionError mismatch (-want +got):\n%s", diff)
			}

			if len(source.selections) != 0 {
				t.Errorf("Cells calls = %d, want 0", len(source.selections))
			}
		})
	}
}

type fakeIndices map[int]app.PriceIndexYear

func (f fakeIndices) PriceIndices(_ context.Context, year int) (app.PriceIndexYear, error) {
	indices, found := f[year]
	if !found {
		return app.PriceIndexYear{Year: year}, nil
	}

	return indices, nil
}

func indexYear(t *testing.T, year, published int) app.PriceIndexYear {
	t.Helper()

	indices := app.PriceIndexYear{
		Year:    year,
		Annual:  map[app.IndexKey]domain.Figure{},
		Monthly: map[app.IndexKey]domain.Figure{},
		Average: map[app.IndexKey]domain.Figure{},
		Core:    map[int]domain.Figure{},
		Tables:  []domain.CatalogEntry{{Code: code(t, cpiAnnual)}},
		ReadAt:  tbilisi(2026, 11, 4),
	}

	for month := 1; month <= 12; month++ {
		for _, region := range domain.Regions() {
			for _, group := range domain.PriceGroups() {
				key := app.IndexKey{Region: region, Group: group, Month: month}
				annual, monthly, average := "..", "..", ".."

				if month <= published {
					annual, monthly, average = "105.9", "100.9", "105.2"
				}

				indices.Annual[key] = figure(t, annual)
				indices.Monthly[key] = figure(t, monthly)

				if group == domain.GroupTotal {
					indices.Average[key] = figure(t, average)
				}
			}
		}

		core := ".."
		if month <= published {
			core = coreRate
		}

		indices.Core[month] = figure(t, core)
	}

	return indices
}

func month(t *testing.T, text string) *domain.Month {
	t.Helper()

	parsed, err := domain.ParseMonth(text)
	if err != nil {
		t.Fatal(err)
	}

	return &parsed
}

// result is the inflation of an answer as text.
type result struct {
	Month, Annual, Monthly, Average, Core string
}

type inflationCase struct {
	name    string
	now     time.Time
	source  fakeIndices
	query   app.InflationQuery
	want    result
	wantErr error
	newest  string
}

func inflationCases(t *testing.T) []inflationCase {
	t.Helper()

	wholeYear := fakeIndices{2026: indexYear(t, 2026, 12)}

	return []inflationCase{
		{
			name: "the newest month", now: tbilisi(2026, 11, 4),
			want: result{october, annual, monthly, average, coreRate},
		},
		{
			name: "a month", now: tbilisi(2026, 11, 4),
			query: app.InflationQuery{Month: month(t, "2026-03")},
			want:  result{"2026-03", annual, monthly, average, coreRate},
		},
		{
			name: "january", now: tbilisi(2027, 1, 3), source: wholeYear,
			want: result{"2026-12", annual, monthly, average, coreRate},
		},
		{
			name: "a city and a group", now: tbilisi(2026, 11, 4),
			query: app.InflationQuery{Region: domain.RegionTbilisi, Group: domain.GroupFood},
			want:  result{october, annual, monthly, "", ""},
		},
		{
			name: "a month with no index", now: tbilisi(2026, 11, 4),
			query:   app.InflationQuery{Month: month(t, "2026-11")},
			wantErr: app.ErrMonthNotPublished, newest: october,
		},
	}
}

// GEOSTAT-SC-013: The default month is the newest published month. A month with no index
// names the newest month. Only the country and the total have the core inflation.
func TestGetInflation(t *testing.T) {
	t.Parallel()

	source := fakeIndices{2026: indexYear(t, 2026, 10)}

	for _, tt := range inflationCases(t) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			indices := source
			if tt.source != nil {
				indices = tt.source
			}

			use := app.NewGetInflation(indices, func() time.Time { return tt.now })

			got, err := use.Get(t.Context(), tt.query)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				var monthErr *app.MonthError
				if !errors.As(err, &monthErr) || monthErr.Newest.String() != tt.newest {
					t.Errorf("Get() error = %v, want the newest month %s", err, tt.newest)
				}

				return
			}

			inflation := got.Inflation
			gotResult := result{
				inflation.Month.String(), inflation.Annual.String(), inflation.Monthly.String(),
				inflation.Average.String(), inflation.Core.String(),
			}

			if diff := cmp.Diff(tt.want, gotResult); diff != "" {
				t.Errorf("inflation mismatch (-want +got):\n%s", diff)
			}

			if len(got.Tables) != 1 {
				t.Errorf("Tables = %d, want 1", len(got.Tables))
			}
		})
	}
}
