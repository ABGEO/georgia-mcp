package geostat_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	geostatadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/adapter/geostat"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/stats/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	geostatclient "github.com/abgeo/georgia-mcp/pkg/geostat"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata    = "../../../../../../pkg/geostat/testdata"
	labour      = "/en/Database/Social Statistics/Labour/"
	cpiFolder   = "/en/Database/Price Statistics/CPI/National Consumer Price Index/"
	wagesRegion = "Social Statistics/Labour/Wages_Region.px"
	labourPath  = "Social Statistics / Labour"

	// clientTimeout ends a stalled call. stallLimit frees the fake server if the client
	// never gives up.
	clientTimeout = 200 * time.Millisecond
	stallLimit    = 5 * time.Second

	annual  = "annual"
	monthly = "monthly"
	average = "average"
	core    = "core"
)

func readAt() time.Time {
	return time.Date(2026, 11, 4, 22, 0, 0, 0, kernel.TbilisiZone())
}

func read(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// keepOnly cuts a recorded folder list to the nodes with the IDs.
func keepOnly(t *testing.T, name string, ids ...string) []byte {
	t.Helper()

	var nodes []geostatclient.Node
	if err := json.Unmarshal(read(t, name), &nodes); err != nil {
		t.Fatal(err)
	}

	kept := []geostatclient.Node{}

	for _, node := range nodes {
		for _, id := range ids {
			if node.ID == id {
				kept = append(kept, node)
			}
		}
	}

	body, err := json.Marshal(kept)
	if err != nil {
		t.Fatal(err)
	}

	return body
}

type answer struct {
	status int
	body   []byte
	// stall holds the answer until the client gives up.
	stall bool
}

// vendor serves an answer for each method and path, and counts the calls of each.
type vendor struct {
	mu      sync.Mutex
	answers map[string][]answer
	calls   map[string]int
}

func newVendor(t *testing.T) (*vendor, *geostatadapter.Adapter, *[]time.Duration) {
	t.Helper()

	fake := &vendor{answers: map[string][]answer{}, calls: map[string]int{}}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	pauses := []time.Duration{}
	client := geostatclient.New(
		vendorhttp.WithBaseURL(server.URL),
		vendorhttp.WithHTTPClient(&http.Client{Timeout: clientTimeout}),
	)
	adapter := geostatadapter.NewAdapter(
		client,
		readAt,
		func(_ context.Context, d time.Duration) error {
			pauses = append(pauses, d)

			return nil
		},
	)

	return fake, adapter, &pauses
}

func (fake *vendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	fake.mu.Lock()
	route := r.Method + " " + r.URL.Path
	call := fake.calls[route]
	fake.calls[route]++
	answers := fake.answers[route]
	fake.mu.Unlock()

	if len(answers) == 0 {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	served := answers[min(call, len(answers)-1)]
	if served.stall {
		select {
		case <-r.Context().Done():
		case <-time.After(stallLimit):
		}

		return
	}

	w.WriteHeader(served.status)
	_, _ = w.Write(served.body)
}

// on sets the answers of a route. The last answer repeats.
func (fake *vendor) on(method, path string, answers ...answer) {
	fake.answers[method+" "+path] = answers
}

func (fake *vendor) count(method, path string) int {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	return fake.calls[method+" "+path]
}

func ok(body []byte) answer {
	return answer{status: http.StatusOK, body: body}
}

func failure() answer {
	return answer{status: http.StatusInternalServerError}
}

func code(t *testing.T, text string) domain.TableCode {
	t.Helper()

	parsed, err := domain.ParseTableCode(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// GEOSTAT-SC-008: A table gives its dimensions, its title, and the update date of its
// folder.
func TestTable(t *testing.T) {
	t.Parallel()

	fake, adapter, _ := newVendor(t)
	fake.on(http.MethodGet, labour, ok(read(t, "folder-labour-en.json")))
	fake.on(http.MethodGet, labour+"Wages_Region.px/", ok(read(t, "wages-region-en.json")))

	got, err := adapter.Table(t.Context(), code(t, wagesRegion), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Table() error = %v", err)
	}

	table := got.Table
	if table.Entry.Title != "Average Monthly Nominal Salary of Employees by Region,Years and Quarters" {
		t.Errorf("Title = %q", table.Entry.Title)
	}

	wantUpdated := time.Date(2026, 10, 8, 19, 17, 47, 0, kernel.TbilisiZone())
	if !table.Entry.Updated.Equal(wantUpdated) ||
		table.Entry.Updated.Location().String() != kernel.TbilisiZone().String() {
		t.Errorf("Updated = %v, want %v", table.Entry.Updated, wantUpdated)
	}

	counts := make([]int, 0, len(table.Dimensions))
	for _, dimension := range table.Dimensions {
		counts = append(counts, len(dimension.Values))
	}

	if diff := cmp.Diff([]int{12, 17, 5}, counts); diff != "" {
		t.Errorf("value counts mismatch (-want +got):\n%s", diff)
	}

	if first := table.Dimensions[0].Values[1]; first != (domain.DimensionValue{Code: "1", Text: "Tbilisi"}) {
		t.Errorf("second region = %+v", first)
	}

	if !got.ReadAt.Equal(readAt()) {
		t.Errorf("ReadAt = %v, want %v", got.ReadAt, readAt())
	}
}

// GEOSTAT-SC-008: A file that the folder does not list, and a folder that answers 404,
// give ErrUnknownTable.
func TestTableUnknown(t *testing.T) {
	t.Parallel()

	fake, adapter, _ := newVendor(t)
	fake.on(http.MethodGet, labour, ok(read(t, "folder-labour-en.json")))

	for _, unknown := range []string{"Social Statistics/Labour/Nope.px", "Nope/Wages_Region.px"} {
		_, err := adapter.Table(t.Context(), code(t, unknown), kernel.LanguageEnglish)
		if !errors.Is(err, app.ErrUnknownTable) {
			t.Errorf("Table(%s) error = %v, want ErrUnknownTable", unknown, err)
		}
	}
}

func serveCPI(t *testing.T, fake *vendor, change func(name string, body []byte) []byte) {
	t.Helper()

	files := map[string]string{
		annual:  "CPI_in_Georgia(same_month_of_the_previous_year=100_orenovani2.px",
		monthly: "CPI_in_Georgia_(previous_month=100)_orenovani2.px",
		average: "CPI_in_Georgia_the_previous_12_month_average=100_orenovani2.px",
		core:    "Core_Inflation-(to_the-same-month-of-the-previous-year).px",
	}

	if change == nil {
		change = func(_ string, body []byte) []byte { return body }
	}

	fake.on(http.MethodGet, cpiFolder, ok(change("folder", read(t, "folder-cpi-en.json"))))

	for name, file := range files {
		metadata := "cpi-" + name + "-en.json"
		cells := "cpi-" + name + "-2026-en.json"

		fake.on(http.MethodGet, cpiFolder+file+"/", ok(change(metadata, read(t, metadata))))
		fake.on(http.MethodPost, cpiFolder+file+"/", ok(change(cells, read(t, cells))))
	}
}

// GEOSTAT-SC-011: The four tables give the indices of 2026 for each region, group, and
// month. A month that Geostat did not publish is unpublished.
func TestPriceIndices(t *testing.T) {
	t.Parallel()

	fake, adapter, _ := newVendor(t)
	serveCPI(t, fake, nil)

	got, err := adapter.PriceIndices(t.Context(), 2026)
	if err != nil {
		t.Fatalf("PriceIndices() error = %v", err)
	}

	october := app.IndexKey{Region: domain.RegionGeorgia, Group: domain.GroupTotal, Month: 10}
	november := october
	november.Month = 11

	values := map[string]string{
		annual:            got.Annual[october].String(),
		monthly:           got.Monthly[october].String(),
		core:              got.Core[10].String(),
		annual + " in XI": got.Annual[november].String(),
	}
	want := map[string]string{
		annual: "105.9", monthly: "100.9", core: "4.0", annual + " in XI": "",
	}

	if diff := cmp.Diff(want, values); diff != "" {
		t.Errorf("indices mismatch (-want +got):\n%s", diff)
	}

	if !got.Average[october].Published() {
		t.Error("the average of October is unpublished")
	}

	food := app.IndexKey{Region: domain.RegionTbilisi, Group: domain.GroupFood, Month: 10}
	if _, found := got.Average[food]; found {
		t.Error("Average holds a group other than the total")
	}

	if !got.Annual[food].Published() || len(got.Annual) != 7*13*12 {
		t.Errorf("Annual holds %d indices, want %d with Tbilisi food", len(got.Annual), 7*13*12)
	}

	if len(got.Tables) != 4 || got.Tables[0].Updated.IsZero() {
		t.Errorf("Tables = %+v, want 4 with update dates", got.Tables)
	}
}

// GEOSTAT-SC-012: A year that the tables do not list gives empty indices and no error.
func TestPriceIndicesOfAYearWithNoTable(t *testing.T) {
	t.Parallel()

	fake, adapter, _ := newVendor(t)
	serveCPI(t, fake, nil)

	got, err := adapter.PriceIndices(t.Context(), 2030)
	if err != nil {
		t.Fatalf("PriceIndices() error = %v", err)
	}

	if len(got.Annual) != 0 || len(got.Core) != 0 || got.Year != 2030 {
		t.Errorf("PriceIndices(2030) = %+v, want empty", got)
	}
}

// GEOSTAT-SC-012: An answer that the adapter cannot read is a failure of Geostat.
func TestPriceIndicesRefusesAChangedAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		file   string
		change func([]byte) []byte
	}{
		{"a renamed group", "cpi-annual-en.json", func(body []byte) []byte {
			return []byte(strings.Replace(string(body), `"Health"`, `"Healthcare"`, 1))
		}},
		{"a missing text", "cpi-average-en.json", func(body []byte) []byte {
			return []byte(strings.Replace(string(body), `,"Zugdidi"`, ``, 1))
		}},
		{"a bad figure", "cpi-monthly-2026-en.json", func(body []byte) []byte {
			return []byte(strings.Replace(string(body), `"100.9"`, `"abc"`, 1))
		}},
		{"a bad date", "folder", func(body []byte) []byte {
			return []byte(strings.Replace(string(body), `"2026-10-01T12:04:07"`, `"yesterday"`, 1))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake, adapter, _ := newVendor(t)
			serveCPI(t, fake, func(name string, body []byte) []byte {
				if name == tt.file {
					return tt.change(body)
				}

				return body
			})

			_, err := adapter.PriceIndices(t.Context(), 2026)
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("PriceIndices() error = %v, want ErrSourceUnavailable", err)
			}
		})
	}
}

// GEOSTAT-SC-012: A failure of the client is a failure of Geostat.
func TestTableReportsAFailure(t *testing.T) {
	t.Parallel()

	fake, adapter, _ := newVendor(t)
	fake.on(http.MethodGet, labour, failure())

	_, err := adapter.Table(t.Context(), code(t, wagesRegion), kernel.LanguageEnglish)
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Table() error = %v, want ErrSourceUnavailable", err)
	}
}

func serveTree(t *testing.T, fake *vendor, labourAnswers ...answer) {
	t.Helper()

	fake.on(http.MethodGet, "/en/Database/", ok(keepOnly(t, "root-en.json", "Social Statistics")))
	fake.on(http.MethodGet, "/en/Database/Social Statistics/",
		ok(keepOnly(t, "folder-social-en.json", "Labour")))
	fake.on(http.MethodGet, labour, labourAnswers...)
}

// GEOSTAT-SC-017: A folder list that hangs is sent again after 2 seconds, at most 3 times.
func TestCatalogRetriesAFolder(t *testing.T) {
	t.Parallel()

	fake, adapter, pauses := newVendor(t)
	serveTree(t, fake, answer{stall: true}, ok(read(t, "folder-labour-en.json")))

	got, err := adapter.Catalog(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}

	if calls := fake.count(http.MethodGet, labour); calls != 2 {
		t.Errorf("calls of Labour = %d, want 2", calls)
	}

	if diff := cmp.Diff([]time.Duration{2 * time.Second}, *pauses); diff != "" {
		t.Errorf("pauses mismatch (-want +got):\n%s", diff)
	}

	if len(got.Entries) != 7 {
		t.Fatalf("entries = %d, want 7", len(got.Entries))
	}

	wages := got.Entries[6]
	if wages.Code.String() != wagesRegion || wages.Folder != labourPath {
		t.Errorf("last entry = %+v", wages)
	}
}

// GEOSTAT-SC-017: A folder that fails 3 times fails the catalog. A query is sent one time.
func TestCatalogFailsAfterThreeAttempts(t *testing.T) {
	t.Parallel()

	fake, adapter, _ := newVendor(t)
	serveTree(t, fake, failure())

	_, err := adapter.Catalog(t.Context(), kernel.LanguageEnglish)
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Catalog() error = %v, want ErrSourceUnavailable", err)
	}

	if calls := fake.count(http.MethodGet, labour); calls != 3 {
		t.Errorf("calls of Labour = %d, want 3", calls)
	}

	query := labour + "Wages_Region.px/"
	fake.on(http.MethodPost, query, failure())

	_, err = adapter.Cells(t.Context(), code(t, wagesRegion), []app.DimensionSelection{
		{Dimension: "Region", Values: []string{"0"}},
	})
	if !errors.Is(err, kernel.ErrSourceUnavailable) || fake.count(http.MethodPost, query) != 1 {
		t.Errorf("Cells() error = %v, calls = %d, want ErrSourceUnavailable and 1 call",
			err, fake.count(http.MethodPost, query))
	}
}

// GEOSTAT-SC-011: The cells of a query carry the key of their value codes.
func TestCells(t *testing.T) {
	t.Parallel()

	fake, adapter, _ := newVendor(t)
	file := "CPI_in_Georgia_(previous_month=100)_orenovani2.px"
	fake.on(http.MethodPost, cpiFolder+file+"/", ok(read(t, "cpi-monthly-query-en.json")))

	selection := []app.DimensionSelection{
		{Dimension: "Region", Values: []string{"0"}},
		{Dimension: "Groups", Values: []string{"0", "1"}},
		{Dimension: "Year", Values: []string{"22"}},
		{Dimension: "month", Values: []string{"9"}},
	}

	got, err := adapter.Cells(t.Context(),
		code(t, "Price Statistics/CPI/National Consumer Price Index/"+file), selection)
	if err != nil {
		t.Fatalf("Cells() error = %v", err)
	}

	figures := map[string]string{}
	for key, figure := range got.Figures {
		figures[key] = figure.String()
	}

	want := map[string]string{
		app.CellKey([]string{"0", "0", "22", "9"}): "100.9",
		app.CellKey([]string{"0", "1", "22", "9"}): "100.0",
	}
	if diff := cmp.Diff(want, figures); diff != "" {
		t.Errorf("figures mismatch (-want +got):\n%s", diff)
	}
}

// node writes one node of a folder list, as Geostat sends it.
func node(id, kind, text string) geostatclient.Node {
	return geostatclient.Node{ID: id, Type: kind, Text: text, Updated: "2026-10-08T19:17:47"}
}

func nodes(t *testing.T, list ...geostatclient.Node) []byte {
	t.Helper()

	body, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// GEOSTAT-SC-008: In Georgian, the catalog and the table give the folder as its path of
// IDs, not as the Georgian names of the folders.
func TestFolderIsThePathInGeorgian(t *testing.T) {
	t.Parallel()

	const georgianLabour = "/ka/Database/Social Statistics/Labour/"

	fake, adapter, _ := newVendor(t)
	fake.on(http.MethodGet, "/ka/Database/",
		ok(nodes(t, node("Social Statistics", "l", "სოციალური სტატისტიკა"))))
	fake.on(http.MethodGet, "/ka/Database/Social Statistics/",
		ok(nodes(t, node("Labour", "l", "შრომის ბაზარი"))))
	fake.on(http.MethodGet, georgianLabour, ok(read(t, "folder-labour-ka.json")))
	fake.on(http.MethodGet, georgianLabour+"Wages_Region.px/", ok(read(t, "wages-region-ka.json")))

	catalog, err := adapter.Catalog(t.Context(), kernel.LanguageGeorgian)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}

	index := slices.IndexFunc(catalog.Entries, func(entry domain.CatalogEntry) bool {
		return entry.Code.String() == wagesRegion
	})
	if index < 0 {
		t.Fatalf("the catalog has no %s", wagesRegion)
	}

	table, err := adapter.Table(t.Context(), code(t, wagesRegion), kernel.LanguageGeorgian)
	if err != nil {
		t.Fatalf("Table() error = %v", err)
	}

	got := []string{catalog.Entries[index].Folder, table.Table.Entry.Folder}
	if diff := cmp.Diff([]string{labourPath, labourPath}, got); diff != "" {
		t.Errorf("folders mismatch (-want +got):\n%s", diff)
	}
}
