package fuel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	testdata      = "../../../../pkg/wissol/testdata"
	getPrices     = "fuel_get_prices"
	now           = "2026-10-19T10:00:00Z"
	regular       = "Euro Regular"
	litre         = "litre"
	gradeReg      = "regular"
	wissolCode    = "wissol"
	rompetrolCode = "rompetrol"
	socarCode     = "socar"
	gradeLPG      = "lpg"
)

type priceOutput struct {
	Product          string `json:"product"`
	Grade            string `json:"grade"`
	Unit             string `json:"unit"`
	Price            string `json:"price"`
	SelfServicePrice string `json:"self_service_price"`
	EffectiveDate    string `json:"effective_date"`
}

type companyOutput struct {
	Company string        `json:"company"`
	Source  string        `json:"source"`
	AsOf    string        `json:"as_of"`
	Stale   bool          `json:"stale"`
	Prices  []priceOutput `json:"prices"`
	Missing []string      `json:"missing"`
}

type pricesOutput struct {
	Meaning   string          `json:"meaning"`
	Companies []companyOutput `json:"companies"`
}

// syncBuffer is a log sink that the server and the test share.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p) //nolint:wrapcheck // A bytes.Buffer never fails.
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// wissolConfig points Wissol at a server that serves the recorded answer.
func wissolConfig(t *testing.T, enabled bool) fuel.Config {
	t.Helper()

	path := filepath.Join(testdata, "fuelprice.json")

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return fuel.Config{Wissol: fuel.WissolConfig{
		Enabled: enabled,
		BaseURL: *base,
		HTTP:    fuel.WissolHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}}
}

func serve(t *testing.T, cfg fuel.Config, logger *slog.Logger) *mcp.ClientSession {
	t.Helper()

	instant, _ := time.Parse(time.RFC3339, now)

	module, err := fuel.NewWithClock(cfg, logger, func() time.Time { return instant })
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = module.Close() })

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, logger)
	if err := registry.Register(module); err != nil {
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

func call(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(
		t.Context(),
		&mcp.CallToolParams{Name: getPrices, Arguments: args},
	)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	return result
}

func decode(t *testing.T, result *mcp.CallToolResult) pricesOutput {
	t.Helper()

	if result.IsError {
		t.Fatalf("result is an error: %+v", result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out pricesOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

func byProduct(company companyOutput) map[string]priceOutput {
	prices := make(map[string]priceOutput, len(company.Prices))
	for _, price := range company.Prices {
		prices[price.Product] = price
	}

	return prices
}

// WISFUEL-SC-016: The answer states the meaning of the two prices, and gives the
// company, its read instant, and each price with its grade, unit, and date.
func TestGetPricesOfWissol(t *testing.T) {
	t.Parallel()

	out := decode(t, call(t, serve(t, wissolConfig(t, true), slog.New(slog.DiscardHandler)), nil))

	wantMeaning := "price: the price in GEL of one unit at a station with an attendant. " +
		"self_service_price: the price in GEL of one unit at a self-service station of the " +
		"same company. A product with no self_service_price has none."
	if out.Meaning != wantMeaning {
		t.Errorf("meaning = %q, want %q", out.Meaning, wantMeaning)
	}

	if len(out.Companies) != 1 {
		t.Fatalf("companies = %+v, want wissol only", out.Companies)
	}

	company := out.Companies[0]

	wantHeader := companyOutput{
		Company: wissolCode,
		Source:  "Wissol",
		AsOf:    "2026-10-19T14:00:00+04:00",
	}
	if header := (companyOutput{
		Company: company.Company, Source: company.Source, AsOf: company.AsOf, Stale: company.Stale,
	}); !cmp.Equal(wantHeader, header) {
		t.Errorf("company = %+v, want %+v", header, wantHeader)
	}

	if len(company.Prices) != 8 {
		t.Errorf("prices = %d, want 8", len(company.Prices))
	}

	prices := byProduct(company)
	want := map[string]priceOutput{
		regular: {
			Product: regular, Grade: gradeReg, Unit: litre, Price: "4.08",
			SelfServicePrice: "3.93", EffectiveDate: "2026-10-08",
		},
		"A1 SUPER 100": {
			Product: "A1 SUPER 100", Grade: "super_100", Unit: litre, Price: "4.79",
			EffectiveDate: "2026-10-17",
		},
		"Wissol Gas": {
			Product: "Wissol Gas", Grade: "cng", Unit: "cubic_metre", Price: "1.64",
			EffectiveDate: "2023-11-07",
		},
	}

	for name, wantPrice := range want {
		if diff := cmp.Diff(wantPrice, prices[name]); diff != "" {
			t.Errorf("%s (-want +got):\n%s", name, diff)
		}
	}

	if first := company.Prices[0].Grade; first != gradeReg {
		t.Errorf("the first price has the grade %q, want regular: the prices sort by grade", first)
	}
}

// WISFUEL-SC-017: The name of a product is in the requested language. Another language
// is a tool error.
func TestGetPricesGivesTheNameInTheLanguage(t *testing.T) {
	t.Parallel()

	session := serve(t, wissolConfig(t, true), slog.New(slog.DiscardHandler))

	tests := map[string]string{"": regular, "en": regular, "ka": "ევრო რეგულარი"}

	for language, want := range tests {
		args := map[string]any{"grades": []string{gradeReg}}
		if language != "" {
			args["language"] = language
		}

		out := decode(t, call(t, session, args))
		if got := out.Companies[0].Prices[0].Product; got != want {
			t.Errorf("language %q: product = %q, want %q", language, got, want)
		}
	}

	result := call(t, session, map[string]any{"language": "ru"})

	text, ok := result.Content[0].(*mcp.TextContent)
	if !result.IsError || !ok ||
		text.Text != `language: "ru" is not a language of this tool. Use ka or en.` {
		t.Errorf("result = %+v, want the tool error of the language", result.Content)
	}
}

// WISFUEL-SC-019: Wissol on gives the tool. Wissol off gives no tool and one info
// record that names the variable.
func TestTheToolNeedsACompany(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		enabled   bool
		wantTools []string
		wantLog   int
	}{
		{name: "on", enabled: true, wantTools: []string{getPrices}},
		{name: "off", enabled: false, wantTools: []string{}, wantLog: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sink := &syncBuffer{}
			logger := slog.New(
				slog.NewJSONHandler(sink, &slog.HandlerOptions{Level: slog.LevelDebug}),
			)
			session := serve(t, wissolConfig(t, tt.enabled), logger)

			list, err := session.ListTools(t.Context(), nil)
			if err != nil {
				t.Fatalf("ListTools() error = %v", err)
			}

			names := make([]string, 0, len(list.Tools))
			for _, tool := range list.Tools {
				names = append(names, tool.Name)
			}

			if diff := cmp.Diff(tt.wantTools, names); diff != "" {
				t.Errorf("tools (-want +got):\n%s", diff)
			}

			if got := strings.Count(
				sink.String(),
				"GEOMCP_FUEL_WISSOL_ENABLED",
			); got != tt.wantLog {
				t.Errorf("records that name the variable = %d, want %d:\n%s", got, tt.wantLog, sink)
			}
		})
	}
}

// rompetrolConfig points Rompetrol at a server that serves a recorded page.
func rompetrolConfig(t *testing.T, enabled bool, page string) fuel.RompetrolConfig {
	t.Helper()

	path := filepath.Join("../../../../pkg/rompetrol/testdata", page)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return fuel.RompetrolConfig{
		Enabled: enabled,
		BaseURL: *base,
		HTTP:    fuel.RompetrolHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}
}

func bothCompanies(t *testing.T, page string) fuel.Config {
	t.Helper()

	cfg := wissolConfig(t, true)
	cfg.Rompetrol = rompetrolConfig(t, true, page)

	return cfg
}

// ROMFUEL-SC-011: The answer holds Rompetrol with no effective date, then Wissol with
// one, and gives the name of a product of Rompetrol in the requested language.
func TestGetPricesOfRompetrol(t *testing.T) {
	t.Parallel()

	session := serve(t, bothCompanies(t, "home.html"), slog.New(slog.DiscardHandler))

	tests := []struct {
		language string
		want     string
	}{
		{language: "en", want: "efix Euro Regular"},
		{language: "ka", want: "efix ევრო რეგულარი"},
	}

	for _, tt := range tests {
		out := decode(t, call(t, session, map[string]any{"language": tt.language}))

		codes := make([]string, 0, len(out.Companies))
		for _, company := range out.Companies {
			codes = append(codes, company.Company)
		}

		if diff := cmp.Diff([]string{rompetrolCode, wissolCode}, codes); diff != "" {
			t.Fatalf("companies (-want +got):\n%s", diff)
		}

		rompetrol := out.Companies[0]
		checkDates(t, rompetrol, out.Companies[1])

		if _, ok := byProduct(rompetrol)[tt.want]; !ok {
			t.Errorf(
				"%s: the prices of Rompetrol lack %q: %+v",
				tt.language,
				tt.want,
				rompetrol.Prices,
			)
		}

		super := byProduct(rompetrol)["efix Super"]
		if tt.language == "en" && (super.Price != "4.65" || super.SelfServicePrice != "4.55") {
			t.Errorf("efix Super = %+v, want 4.65 and 4.55", super)
		}
	}
}

// checkDates checks the source, the read instant, and the absent dates of Rompetrol,
// and the dates of Wissol.
func checkDates(t *testing.T, rompetrol, wissol companyOutput) {
	t.Helper()

	asOf, _ := time.Parse(time.RFC3339, rompetrol.AsOf)
	instant, _ := time.Parse(time.RFC3339, now)

	if rompetrol.Source != "Rompetrol" || !asOf.Equal(instant) || len(rompetrol.Prices) != 5 {
		t.Errorf(
			"rompetrol = %s at %s with %d prices, want Rompetrol at %s with 5",
			rompetrol.Source, rompetrol.AsOf, len(rompetrol.Prices), now,
		)
	}

	for _, price := range rompetrol.Prices {
		if price.EffectiveDate != "" {
			t.Errorf("%s effective_date = %q, want none", price.Product, price.EffectiveDate)
		}
	}

	for _, price := range wissol.Prices {
		if price.EffectiveDate == "" {
			t.Errorf("%s has no effective_date", price.Product)
		}
	}
}

// ROMFUEL-SC-012: The description tells which company publishes no effective date, and
// the schema of the output does not require effective_date.
func TestTheToolTellsWhoPublishesNoDate(t *testing.T) {
	t.Parallel()

	session := serve(t, bothCompanies(t, "home.html"), slog.New(slog.DiscardHandler))

	list, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	tool := list.Tools[0]

	const companies = "rompetrol (Rompetrol, publishes no effective date), wissol (Wissol)"
	if !strings.Contains(tool.Description, companies) {
		t.Errorf("description = %q, want it to hold %q", tool.Description, companies)
	}

	schema, err := json.Marshal(tool.OutputSchema)
	if err != nil {
		t.Fatal(err)
	}

	var price struct {
		Defs map[string]struct {
			Required []string `json:"required"`
		} `json:"$defs"`
		Properties struct {
			Companies struct {
				Items struct {
					Properties struct {
						Prices struct {
							Items struct {
								Required []string `json:"required"`
							} `json:"items"`
						} `json:"prices"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"companies"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &price); err != nil {
		t.Fatal(err)
	}

	required := price.Properties.Companies.Items.Properties.Prices.Items.Required
	if len(required) == 0 {
		t.Fatalf("the schema of a price lists no required field: %s", schema)
	}

	for _, field := range required {
		if field == "effective_date" {
			t.Errorf("the schema requires effective_date: %v", required)
		}
	}
}

// ROMFUEL-SC-013: A page of Rompetrol with no price table makes Rompetrol unavailable,
// and the answer holds Wissol.
func TestGetPricesWithNoTableOfRompetrol(t *testing.T) {
	t.Parallel()

	session := serve(t, bothCompanies(t, "home-no-table.html"), slog.New(slog.DiscardHandler))

	result := call(t, session, map[string]any{})
	out := decode(t, result)

	if len(out.Companies) != 1 || out.Companies[0].Company != wissolCode {
		t.Errorf("companies = %+v, want wissol only", out.Companies)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var unavailable struct {
		Unavailable []struct {
			Company string `json:"company"`
		} `json:"unavailable"`
	}
	if err := json.Unmarshal(raw, &unavailable); err != nil {
		t.Fatal(err)
	}

	if len(unavailable.Unavailable) != 1 || unavailable.Unavailable[0].Company != rompetrolCode {
		t.Errorf("unavailable = %+v, want rompetrol", unavailable.Unavailable)
	}
}

// ROMFUEL-SC-016: Rompetrol off logs the variable, and a request for it is a tool error.
func TestRompetrolOff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	logger := slog.New(slog.NewJSONHandler(sink, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := wissolConfig(t, true)
	cfg.Rompetrol = rompetrolConfig(t, false, "home.html")

	session := serve(t, cfg, logger)

	if !strings.Contains(sink.String(), `"msg":"Rompetrol is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_FUEL_ROMPETROL_ENABLED"`) {
		t.Errorf("the log lacks the record of Rompetrol off:\n%s", sink)
	}

	result := call(t, session, map[string]any{"companies": []string{rompetrolCode}})
	if !result.IsError {
		t.Fatalf("result = %+v, want a tool error", result.StructuredContent)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, `"rompetrol" is not a fuel company of this server`) {
		t.Errorf("result = %+v, want the error of an unknown company", result.Content)
	}
}

// socarConfig points SOCAR at a server that answers with a status and a recorded body.
func socarConfig(t *testing.T, enabled bool, answer string, status int) fuel.SOCARConfig {
	t.Helper()

	path := filepath.Join("../../../../pkg/socar/testdata", answer)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return fuel.SOCARConfig{
		Enabled: enabled,
		BaseURL: *base,
		HTTP:    fuel.SOCARHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}
}

func threeCompanies(t *testing.T) fuel.Config {
	t.Helper()

	cfg := bothCompanies(t, "home.html")
	cfg.SOCAR = socarConfig(t, true, "current-prices.json", http.StatusOK)

	return cfg
}

// SOCFUEL-SC-009: The grade lpg gives LPG of SOCAR, and names lpg as missing at the
// other companies.
func TestGetPricesOfLPG(t *testing.T) {
	t.Parallel()

	session := serve(t, threeCompanies(t), slog.New(slog.DiscardHandler))

	out := decode(t, call(t, session, map[string]any{"grades": []string{gradeLPG}}))

	if len(out.Companies) != 3 {
		t.Fatalf("companies = %+v, want 3", out.Companies)
	}

	for _, company := range out.Companies {
		if company.Company != socarCode {
			if len(company.Prices) != 0 || !slices.Equal(company.Missing, []string{gradeLPG}) {
				t.Errorf("%s = %+v, want no price and lpg missing", company.Company, company)
			}

			continue
		}

		want := []priceOutput{{Product: "LPG", Grade: gradeLPG, Unit: litre, Price: "2.09"}}
		if diff := cmp.Diff(want, company.Prices); diff != "" {
			t.Errorf("prices of SOCAR (-want +got):\n%s", diff)
		}
	}
}

// SOCFUEL-SC-010: The description lists lpg and SOCAR with no effective date, and the
// schema of grades names lpg.
func TestTheToolListsLPGAndSOCAR(t *testing.T) {
	t.Parallel()

	session := serve(t, threeCompanies(t), slog.New(slog.DiscardHandler))

	list, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	tool := list.Tools[0]

	for _, want := range []string{
		"lpg (litre)",
		"rompetrol (Rompetrol, publishes no effective date), " +
			"socar (SOCAR, publishes no effective date), wissol (Wissol)",
	} {
		if !strings.Contains(tool.Description, want) {
			t.Errorf("description = %q, want it to hold %q", tool.Description, want)
		}
	}

	schema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(schema), "cng, lpg, other") {
		t.Errorf("the input schema does not name lpg: %s", schema)
	}
}

// SOCFUEL-SC-012: An error of SOCAR makes SOCAR unavailable, and the answer holds Wissol.
func TestGetPricesWithAnErrorOfSOCAR(t *testing.T) {
	t.Parallel()

	cfg := wissolConfig(t, true)
	cfg.SOCAR = socarConfig(t, true, "error.json", http.StatusBadRequest)

	result := call(t, serve(t, cfg, slog.New(slog.DiscardHandler)), map[string]any{})
	out := decode(t, result)

	if len(out.Companies) != 1 || out.Companies[0].Company != wissolCode {
		t.Errorf("companies = %+v, want wissol only", out.Companies)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(raw), `"unavailable":[{"company":"socar","source":"SOCAR"}]`) {
		t.Errorf("the answer does not name SOCAR as unavailable: %s", raw)
	}
}

// SOCFUEL-SC-015: SOCAR off logs the variable, and a request for it is a tool error.
func TestSOCAROff(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	logger := slog.New(slog.NewJSONHandler(sink, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := wissolConfig(t, true)
	cfg.SOCAR = socarConfig(t, false, "current-prices.json", http.StatusOK)

	session := serve(t, cfg, logger)

	if !strings.Contains(sink.String(), `"msg":"SOCAR is off"`) ||
		!strings.Contains(sink.String(), `"variable":"GEOMCP_FUEL_SOCAR_ENABLED"`) {
		t.Errorf("the log lacks the record of SOCAR off:\n%s", sink)
	}

	result := call(t, session, map[string]any{"companies": []string{socarCode}})
	if !result.IsError {
		t.Fatalf("result = %+v, want a tool error", result.StructuredContent)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, `"socar" is not a fuel company of this server`) {
		t.Errorf("result = %+v, want the error of an unknown company", result.Content)
	}
}
