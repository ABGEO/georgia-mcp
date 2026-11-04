package tool_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	wagesRegion = "Social Statistics/Labour/Wages_Region.px"
	cpiAnnual   = "Price Statistics/CPI/National Consumer Price Index/" +
		"CPI_in_Georgia(same_month_of_the_previous_year=100_orenovani2.px"
	keyCode     = "code"
	keyLanguage = "language"
	keyQuery    = "query"
	keyValues   = "values"
	keyRegion   = "region"
	keyMonth    = "month"
	georgia     = "Georgia"
	salary      = "salary"
)

func readAt() time.Time {
	return time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)
}

func updated() time.Time {
	return time.Date(2026, 10, 8, 19, 17, 47, 0, kernel.TbilisiZone())
}

func code(t *testing.T, text string) domain.TableCode {
	t.Helper()

	parsed, err := domain.ParseTableCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func figure(t *testing.T, text string) domain.Figure {
	t.Helper()

	parsed, err := domain.ParseFigure(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

type fakeCatalog struct {
	entries []domain.CatalogEntry
}

func (f fakeCatalog) Catalog(context.Context, kernel.Language) (app.Catalog, error) {
	return app.Catalog{Entries: f.entries, ReadAt: readAt()}, nil
}

// fakeTables holds one table with a region and a month. The cell of Tbilisi in month II is
// unpublished.
type fakeTables struct {
	table domain.Table
}

func (f fakeTables) Table(
	_ context.Context,
	tableCode domain.TableCode,
	_ kernel.Language,
) (app.TableDescription, error) {
	if tableCode != f.table.Entry.Code {
		return app.TableDescription{}, app.ErrUnknownTable
	}

	return app.TableDescription{Table: f.table, ReadAt: readAt()}, nil
}

func (f fakeTables) Cells(
	context.Context,
	domain.TableCode,
	[]app.DimensionSelection,
) (app.CellSet, error) {
	unpublished, _ := domain.ParseFigure("..")
	annual, _ := domain.ParseFigure("105.9")

	return app.CellSet{
		Figures: map[string]domain.Figure{
			app.CellKey([]string{"0", "0"}): annual,
			app.CellKey([]string{"0", "1"}): unpublished,
		},
		ReadAt: readAt(),
	}, nil
}

type fakeIndices struct {
	indices app.PriceIndexYear
}

func (f fakeIndices) PriceIndices(_ context.Context, year int) (app.PriceIndexYear, error) {
	if year != f.indices.Year {
		return app.PriceIndexYear{Year: year}, nil
	}

	return f.indices, nil
}

func cpiTable(t *testing.T) domain.Table {
	t.Helper()

	return domain.Table{
		Entry: domain.CatalogEntry{
			Code: code(t, cpiAnnual), Title: "CPI (same month of the previous year=100)",
			Folder: "Price Statistics / CPI / National Consumer Price Index", Updated: updated(),
		},
		Dimensions: []domain.Dimension{
			{Code: "Region", Text: "Region", Values: []domain.DimensionValue{
				{Code: "0", Text: georgia},
			}},
			{Code: keyMonth, Text: keyMonth, Values: []domain.DimensionValue{
				{Code: "0", Text: "I"}, {Code: "1", Text: "II"},
			}},
		},
	}
}

func indices(t *testing.T) app.PriceIndexYear {
	t.Helper()

	year := app.PriceIndexYear{
		Year:    2026,
		Annual:  map[app.IndexKey]domain.Figure{},
		Monthly: map[app.IndexKey]domain.Figure{},
		Average: map[app.IndexKey]domain.Figure{},
		Core:    map[int]domain.Figure{10: figure(t, "4.0")},
		ReadAt:  readAt(),
	}

	for month := 1; month <= 10; month++ {
		key := app.IndexKey{Region: domain.RegionGeorgia, Group: domain.GroupTotal, Month: month}
		year.Annual[key] = figure(t, "105.9")
		year.Monthly[key] = figure(t, "100.9")
		year.Average[key] = figure(t, "105.2")
	}

	for range 4 {
		year.Tables = append(year.Tables, cpiTable(t).Entry)
	}

	return year
}

func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()

	wages := cpiTable(t)
	wages.Entry.Code = code(t, wagesRegion)
	wages.Entry.Title = "Average Monthly Nominal Salary of Employees by Region,Years and Quarters"
	wages.Entry.Folder = "Social Statistics / Labour"

	catalog := fakeCatalog{entries: []domain.CatalogEntry{wages.Entry, cpiTable(t).Entry}}
	tables := fakeTables{table: cpiTable(t)}
	now := func() time.Time { return time.Date(2026, 11, 4, 12, 0, 0, 0, kernel.TbilisiZone()) }

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		SearchTables:  app.NewSearchTables(catalog),
		DescribeTable: app.NewDescribeTable(tables),
		GetCells:      app.NewGetCells(tables),
		GetInflation:  app.NewGetInflation(fakeIndices{indices: indices(t)}, now),
	}); err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = registry.RunStdio(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).
		Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

func callTool(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	return result
}

// call returns the structured output of a tool as a map of JSON values.
func call(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) map[string]any {
	t.Helper()

	result := callTool(t, session, name, args)
	if result.IsError {
		t.Fatalf("%s() = %+v, want a result", name, result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

func errorText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()

	result := callTool(t, session, name, args)
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("%s() = %+v, want a tool error", name, result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("%s() content = %T, want text", name, result.Content[0])
	}

	return text.Text
}

// refusal is one input that a tool refuses, and the message of the refusal.
type refusal struct {
	name string
	args map[string]any
	want string
}

func refused(t *testing.T, session *mcp.ClientSession, name string, cases []refusal) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := errorText(t, session, name, tt.args); got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
}

// GEOSTAT-SC-014: A search gives the source, the instant in Tbilisi, the total, and the
// tables. A bad language and a bad limit give the messages of the tool.
func TestSearchTables(t *testing.T) {
	t.Parallel()

	session := connect(t)

	out := call(t, session, "stats_search_tables", map[string]any{keyQuery: salary})

	if out["source"] != "Geostat" || out["as_of"] != "2026-11-04T12:00:00+04:00" ||
		out["total"] != 1.0 {
		t.Errorf("output = %v", out)
	}

	tables, _ := out["tables"].([]any)
	if len(tables) != 1 {
		t.Fatalf("tables = %v, want 1", out["tables"])
	}

	want := map[string]any{
		"code": wagesRegion, "updated": "2026-10-08",
		"title":  "Average Monthly Nominal Salary of Employees by Region,Years and Quarters",
		"folder": "Social Statistics / Labour",
	}
	if diff := cmp.Diff(want, tables[0]); diff != "" {
		t.Errorf("table mismatch (-want +got):\n%s", diff)
	}
}

// GEOSTAT-SC-014: A bad language, limit, or query gives the message of the tool.
func TestSearchTablesRefusesABadInput(t *testing.T) {
	t.Parallel()

	refused(t, connect(t), "stats_search_tables", []refusal{
		{
			name: "a language", args: map[string]any{keyQuery: salary, keyLanguage: "ru"},
			want: `language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{
			name: "a limit", args: map[string]any{keyQuery: salary, "limit": 500},
			want: "limit: 500 is out of range. Use 1 to 100.",
		},
		{
			name: "a query", args: map[string]any{keyQuery: "  "},
			want: "query: give 1 to 100 characters.",
		},
	})
}

// GEOSTAT-SC-015: A table gives its dimensions and its update date. The cells give a
// figure as a decimal string, and null for an unpublished figure. An unknown code gives
// the message of the tool.
func TestGetTableAndCells(t *testing.T) {
	t.Parallel()

	session := connect(t)

	table := call(t, session, "stats_get_table", map[string]any{keyCode: cpiAnnual})
	dimensions, _ := table["dimensions"].([]any)

	if table["updated"] != "2026-10-08" || len(dimensions) != 2 || table["source"] != "Geostat" {
		t.Errorf("table = %v", table)
	}

	cells := call(t, session, "stats_get_cells", map[string]any{keyCode: cpiAnnual})

	want := []any{
		map[string]any{keyValues: []any{georgia, "I"}, "figure": "105.9"},
		map[string]any{keyValues: []any{georgia, "II"}, "figure": nil},
	}
	if diff := cmp.Diff(want, cells["cells"]); diff != "" {
		t.Errorf("cells mismatch (-want +got):\n%s", diff)
	}

	if inner, _ := cells["table"].(map[string]any); inner["source"] != nil ||
		inner["code"] != cpiAnnual {
		t.Errorf("table of the cells = %v, want the code and no source", inner)
	}

	refused(t, session, "stats_get_table", []refusal{
		{
			name: "a code with no table",
			args: map[string]any{keyCode: "Social Statistics/Labour/Nope.px"},
			want: `code: Geostat has no table "Social Statistics/Labour/Nope.px". ` +
				"Call stats_search_tables for the codes.",
		},
		{
			name: "a bad code", args: map[string]any{keyCode: "x"},
			want: `code: "x" is not a table code. Call stats_search_tables for the codes.`,
		},
	})

	selection := map[string]any{keyCode: cpiAnnual, "selection": []any{
		map[string]any{"dimension": "Town", keyValues: []any{"0"}},
	}}
	wantSelection := `selection: the table has no dimension "Town". Use Region, month.`

	if got := errorText(t, session, "stats_get_cells", selection); got != wantSelection {
		t.Errorf("message = %q, want %q", got, wantSelection)
	}
}

// GEOSTAT-SC-016: The inflation with no input is the newest month of the country, in
// percent. Georgian names the region in Georgian. An unknown region gives the message
// of the tool.
func TestGetInflation(t *testing.T) {
	t.Parallel()

	session := connect(t)

	out := call(t, session, "stats_get_inflation", map[string]any{})

	got := map[string]any{
		keyMonth: out[keyMonth], "annual": out["annual"], "monthly": out["monthly"],
		"average": out["average"], "core": out["core"], keyRegion: out[keyRegion],
	}
	want := map[string]any{
		keyMonth: "2026-10", "annual": "5.9", "monthly": "0.9", "average": "5.2", "core": "4.0",
		keyRegion: map[string]any{"code": "georgia", "name": georgia},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("inflation mismatch (-want +got):\n%s", diff)
	}

	if tables, _ := out["tables"].([]any); len(tables) != 4 {
		t.Errorf("tables = %d, want 4", len(tables))
	}

	georgian := call(t, session, "stats_get_inflation", map[string]any{keyLanguage: "ka"})
	if region, _ := georgian[keyRegion].(map[string]any); region["name"] != "საქართველო" {
		t.Errorf("region in ka = %v, want საქართველო", georgian[keyRegion])
	}
}

// GEOSTAT-SC-016: An unknown region, a bad month, and a month with no index give the
// message of the tool.
func TestGetInflationRefusesABadInput(t *testing.T) {
	t.Parallel()

	refused(t, connect(t), "stats_get_inflation", []refusal{
		{
			name: "a region", args: map[string]any{keyRegion: "rustavi"},
			want: `region: "rustavi" is not a region of the index. Use georgia, tbilisi, ` +
				"kutaisi, batumi, gori, telavi, or zugdidi.",
		},
		{
			name: "a bad month", args: map[string]any{keyMonth: "October"},
			want: "month: give the month as YYYY-MM, for example 2026-10.",
		},
		{
			name: "a month with no index", args: map[string]any{keyMonth: "2026-11"},
			want: "month: Geostat published no consumer price index for 2026-11. " +
				"The newest month is 2026-10.",
		},
	})
}
