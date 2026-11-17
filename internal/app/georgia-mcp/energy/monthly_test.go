package energy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/energy"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	gnercTestdata = "../../../../pkg/gnerc/testdata/"
	getBalance    = "energy_get_balance"
	listPrices    = "energy_list_prices"
	gnercName     = "Georgian National Energy and Water Supply Regulatory Commission (GNERC)"
	gnercChanged  = "2026-10-21T13:45:18+04:00"
	newestMonth   = "2026-09"
	gnercOutage   = "GNERC did not answer. Try again in a few minutes."
	keyMonth      = "month"
	keyYear       = "year"
	keyFrom       = "from"
	keyTo         = "to"
	march2026     = "2026-03"
)

// fakeGNERC serves the recorded home page and workbooks, or the status for every path,
// and counts the calls of each path.
type fakeGNERC struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  map[string]int
}

func newFakeGNERC(t *testing.T, status int) *fakeGNERC {
	t.Helper()

	files := map[string]string{
		"/": "home.html",
		"2015 - 2026 ფაქტიური ბალანსი.xlsx":                         "balance.xlsx",
		"საბალანსო ელექტროენერგიის ფასები 2015 - 2026 წლები.xlsx":   "balancing.xlsx",
		"გარანტირებული სიმძლავრის საფასური  2015 - 2026 წლები.xlsx": "capacity-fee.xlsx",
		"დერეგულირებული სადგურების ფასები 2015 - 2026 წლები.xlsx":   "deregulated.xlsx",
	}

	bodies := map[string][]byte{}

	for name, file := range files {
		body, err := os.ReadFile(gnercTestdata + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		if name != "/" {
			name = "/excel/" + url.PathEscape(name)
		}

		bodies[name] = body
	}

	fake := &fakeGNERC{calls: map[string]int{}}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.calls[r.URL.EscapedPath()]++
		fake.mu.Unlock()

		body, found := bodies[r.URL.EscapedPath()]

		switch {
		case status != http.StatusOK:
			w.WriteHeader(status)
		case !found:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2026 09:45:18 GMT")
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

// counts returns the calls of the home page and the calls of the workbooks.
func (f *fakeGNERC) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	workbooks := 0

	for path, count := range f.calls {
		if path != "/" {
			workbooks += count
		}
	}

	return f.calls["/"], workbooks
}

func (f *fakeGNERC) config(t *testing.T) energy.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return energy.Config{GNERC: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 20},
	}}
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
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}

	return result
}

// outputOf decodes the structured content of a result that is not an error.
func outputOf[T any](t *testing.T, session *mcp.ClientSession, name string, args map[string]any) T {
	t.Helper()

	result := callTool(t, session, name, args)
	if result.IsError {
		t.Fatalf("CallTool(%s, %v) = %+v, want a result", name, args, result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

func toolErrorText(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) string {
	t.Helper()

	result := callTool(t, session, name, args)
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool(%s, %v) = %+v, want a tool error", name, args, result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool() content = %T, want text", result.Content[0])
	}

	return text.Text
}

func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(tools.Tools))

	for _, tool := range tools.Tools {
		if !readOnly(tool.Annotations) {
			t.Errorf("%s is not read-only", tool.Name)
		}

		names = append(names, tool.Name)
	}

	slices.Sort(names)

	return names
}

// GNERCMON-SC-021: Each switch decides the tools of its vendor.
func TestEachSwitchDecidesItsTools(t *testing.T) {
	t.Parallel()

	gseOnly := serve(t, newFakeGSE(t, http.StatusOK).config(t), discard(), at(1, 30))
	if got := toolNames(t, gseOnly); !slices.Equal(got, []string{getGrid}) {
		t.Errorf("tools with GSE only = %v, want %s", got, getGrid)
	}

	gnercOnly := serve(t, newFakeGNERC(t, http.StatusOK).config(t), discard(), at(1, 30))
	if got := toolNames(t, gnercOnly); !slices.Equal(got, []string{getBalance, listPrices}) {
		t.Errorf("tools with GNERC only = %v, want %s and %s", got, getBalance, listPrices)
	}
}

type balanceOutput struct {
	Source      string `json:"source"`
	AsOf        string `json:"as_of"`
	Period      string `json:"period"`
	Months      int    `json:"months"`
	NewestMonth string `json:"newest_month"`
	Unit        string `json:"unit"`
	Generation  struct {
		Total string  `json:"total"`
		Solar *string `json:"solar"`
	} `json:"generation"`
	Import struct {
		Russia string `json:"russia"`
	} `json:"import"`
	Export struct {
		Turkey string `json:"turkey"`
	} `json:"export"`
	Consumption struct {
		Suppliers *struct {
			FreeMarket *string `json:"free_market"`
		} `json:"suppliers"`
	} `json:"consumption"`
	DomesticConsumption string `json:"domestic_consumption"`
}

// GNERCMON-SC-022: energy_get_balance reads the recorded balance through the whole
// context.
func TestGetBalanceReadsGNERC(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeGNERC(t, http.StatusOK).config(t), discard(), at(1, 30))

	march := outputOf[balanceOutput](t, session, getBalance, map[string]any{keyMonth: march2026})
	if march.Source != gnercName || march.AsOf != gnercChanged || march.Period != march2026 ||
		march.Months != 1 || march.NewestMonth != newestMonth || march.Unit != "million kWh" {
		t.Errorf("2026-03 = %+v, want GNERC, %s, 2026-03, 1 month, newest 2026-09, million kWh",
			march, gnercChanged)
	}

	checkMarch(t, march)

	year := outputOf[balanceOutput](t, session, getBalance, map[string]any{keyYear: 2026})
	if year.Period != "2026" || year.Months != 9 || year.Generation.Total != "10308.382" {
		t.Errorf("2026 = %s, %d months, %s; want 2026, 9, 10308.382",
			year.Period, year.Months, year.Generation.Total)
	}

	checkNoNewItems(
		t,
		outputOf[balanceOutput](t, session, getBalance, map[string]any{keyMonth: "2022-03"}),
	)
}

func checkMarch(t *testing.T, march balanceOutput) {
	t.Helper()

	if march.Generation.Total != "1102.668" || march.Generation.Solar == nil ||
		*march.Generation.Solar != "1.314" || march.Import.Russia != "155.484" ||
		march.Export.Turkey != "0.000" || march.DomesticConsumption != "1326.633" {
		t.Errorf("2026-03 values = %+v, want the recorded values", march)
	}
}

// checkNoNewItems checks that 2022-03 holds neither solar nor the free market suppliers.
func checkNoNewItems(t *testing.T, old balanceOutput) {
	t.Helper()

	if old.Generation.Solar != nil ||
		(old.Consumption.Suppliers != nil && old.Consumption.Suppliers.FreeMarket != nil) {
		t.Errorf("2022-03 = %+v, want no solar and no free market", old)
	}
}

type pricesOutput struct {
	Source      string `json:"source"`
	AsOf        string `json:"as_of"`
	NewestMonth string `json:"newest_month"`
	Unit        string `json:"unit"`
	Months      []struct {
		Month       string `json:"month"`
		Balancing   string `json:"balancing"`
		CapacityFee string `json:"capacity_fee"`
		Deregulated string `json:"deregulated"`
	} `json:"months"`
}

// GNERCMON-SC-023: energy_list_prices reads the recorded prices through the whole context,
// and ends the period at the newest month.
func TestListPricesReadsGNERC(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeGNERC(t, http.StatusOK).config(t), discard(), at(1, 30))

	out := outputOf[pricesOutput](
		t,
		session,
		listPrices,
		map[string]any{keyFrom: "2026-07", keyTo: "2026-12"},
	)
	if out.Source != gnercName || out.AsOf != gnercChanged || out.Unit != "tetri per kWh" ||
		out.NewestMonth != newestMonth || len(out.Months) != 3 {
		t.Fatalf("prices = %+v, want GNERC, tetri per kWh, newest 2026-09, 3 months", out)
	}

	last := out.Months[2]
	if last.Month != newestMonth || last.Balancing != "16.068" || last.CapacityFee != "1.607" ||
		last.Deregulated != "11.655" {
		t.Errorf("last month = %+v, want 2026-09 with 16.068, 1.607, 11.655", last)
	}
}

// GNERCMON-SC-024: Each bad input of 4.5 gives its message.
func TestTheToolsRefuseABadPeriod(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeGNERC(t, http.StatusOK).config(t), discard(), at(1, 30))

	tests := []struct {
		tool string
		args map[string]any
		want string
	}{
		{
			getBalance,
			map[string]any{keyMonth: "March"},
			`month: "March" is not a month. Use YYYY-MM, for example 2026-03.`,
		},
		{
			getBalance,
			map[string]any{keyMonth: march2026, keyYear: 2026},
			"month and year: give one of them, or neither for the newest published month.",
		},
		{
			getBalance,
			map[string]any{keyMonth: "2021-12"},
			"month: 2021-12 is outside the balance of GNERC. Give a month from 2022-01 to 2026-09.",
		},
		{
			getBalance,
			map[string]any{keyYear: 2021},
			"year: 2021 is outside the balance of GNERC. Give a year from 2022 to 2026.",
		},
		{
			listPrices,
			map[string]any{keyFrom: "2014-12"},
			"from: 2014-12 is outside the prices of GNERC. Give a month from 2015-01 to 2026-09.",
		},
		{
			listPrices,
			map[string]any{keyFrom: "2025-06", keyTo: "2025-01"},
			"to: 2025-01 is before from, 2025-06. Give a period that ends on or after its start.",
		},
		{
			listPrices,
			map[string]any{keyFrom: "2020-01", keyTo: "2025-12"},
			"from and to: the period holds 72 months. Give at most 60.",
		},
	}

	for _, tt := range tests {
		if got := toolErrorText(t, session, tt.tool, tt.args); got != tt.want {
			t.Errorf("%s(%v) = %q, want %q", tt.tool, tt.args, got, tt.want)
		}
	}
}

// GNERCMON-SC-025: A failing GNERC with an empty cache gives the message of the outage.
func TestAFailingGNERC(t *testing.T) {
	t.Parallel()

	session := serve(
		t,
		newFakeGNERC(t, http.StatusInternalServerError).config(t),
		discard(),
		at(1, 30),
	)

	for _, name := range []string{getBalance, listPrices} {
		if got := toolErrorText(t, session, name, nil); got != gnercOutage {
			t.Errorf("%s() = %q, want %q", name, got, gnercOutage)
		}
	}
}

// GNERCMON-SC-026: Two calls of each tool read the home page and each workbook once.
func TestCallsShareOneBook(t *testing.T) {
	t.Parallel()

	balanceFake := newFakeGNERC(t, http.StatusOK)
	session := serve(t, balanceFake.config(t), discard(), at(1, 30))

	for range 2 {
		outputOf[balanceOutput](t, session, getBalance, nil)
	}

	if home, workbooks := balanceFake.counts(); home != 1 || workbooks != 1 {
		t.Errorf(
			"balance calls = %d of the home page, %d of the workbooks; want 1 and 1",
			home,
			workbooks,
		)
	}

	pricesFake := newFakeGNERC(t, http.StatusOK)
	session = serve(t, pricesFake.config(t), discard(), at(1, 30))

	for range 2 {
		outputOf[pricesOutput](t, session, listPrices, nil)
	}

	if home, workbooks := pricesFake.counts(); home != 1 || workbooks != 3 {
		t.Errorf(
			"price calls = %d of the home page, %d of the workbooks; want 1 and 3",
			home,
			workbooks,
		)
	}
}
