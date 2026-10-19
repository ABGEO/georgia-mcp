package tool_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const getPrices = "fuel_get_prices"

type fakePrices struct {
	prices app.Prices
	err    error
}

func (f fakePrices) Get(context.Context, app.PricesQuery) (app.Prices, error) {
	return f.prices, f.err
}

func (fakePrices) Companies() []domain.Company {
	return []domain.Company{wissol()}
}

func wissol() domain.Company {
	return domain.Company{Code: "wissol", Name: "Wissol", PublishesEffectiveDate: true}
}

func connect(t *testing.T, prices fakePrices) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, prices); err != nil {
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

func errorText(t *testing.T, session *mcp.ClientSession, args map[string]any) string {
	t.Helper()

	result, err := session.CallTool(
		t.Context(),
		&mcp.CallToolParams{Name: getPrices, Arguments: args},
	)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("result = %+v, want a tool error", result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T, want text", result.Content[0])
	}

	return text.Text
}

// WISFUEL-SC-018: A grade or a company that the tool does not know gives a message
// with the accepted values.
func TestGetPricesNamesTheAcceptedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		args map[string]any
		want string
	}{
		{
			name: "unknown grade",
			args: map[string]any{"grades": []string{"lpg"}},
			want: `grades: "lpg" is not a grade. ` +
				"Use one of: regular, premium, super, super_100, diesel, cng, other.",
		},
		{
			name: "unknown company",
			err:  &app.UnknownCompanyError{Code: "gulf", Known: []domain.CompanyCode{"wissol"}},
			args: map[string]any{"companies": []string{"gulf"}},
			want: `companies: "gulf" is not a fuel company of this server. Use one of: wissol.`,
		},
		{
			name: "not a company code",
			args: map[string]any{"companies": []string{"Wissol"}},
			want: `companies: "Wissol" is not a company code. Use one of: wissol.`,
		},
		{
			name: "unknown language",
			args: map[string]any{"language": "ru"},
			want: `language: "ru" is not a language of this tool. Use ka or en.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := connect(t, fakePrices{err: tt.err})
			if got := errorText(t, session, tt.args); got != tt.want {
				t.Errorf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

// ROMFUEL-SC-010: A price with the zero date has no key effective_date. A price with a
// date has it.
func TestGetPricesLeavesOutTheZeroDate(t *testing.T) {
	t.Parallel()

	changed, err := kernel.ParseDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}

	priceOn := func(code string, effective kernel.Date) domain.ProductPrice {
		price, err := domain.NewProductPrice(
			code,
			domain.ProductName{English: code},
			domain.GradeRegular,
			decimal.RequireFromString("4.02"),
			decimal.NullDecimal{},
			effective,
		)
		if err != nil {
			t.Fatal(err)
		}

		return price
	}

	session := connect(t, fakePrices{prices: app.Prices{Companies: []app.CompanyPrices{
		{
			Company: domain.Company{Code: "rompetrol", Name: "Rompetrol"},
			Prices:  []domain.ProductPrice{priceOn("undated", kernel.Date{})},
		},
		{
			Company: wissol(),
			Prices:  []domain.ProductPrice{priceOn("dated", changed)},
		},
	}}})

	raw, prices := pricesOf(t, session)

	if _, ok := prices[0]["effective_date"]; ok {
		t.Errorf("the undated price = %v, want no effective_date", prices[0])
	}

	if got := prices[1]["effective_date"]; got != "2026-10-08" {
		t.Errorf("effective_date = %v, want 2026-10-08", got)
	}

	if strings.Contains(string(raw), `"effective_date":""`) {
		t.Errorf("the output holds an empty effective_date: %s", raw)
	}
}

// pricesOf calls the tool with no input, and returns the raw output and the first price
// of each company as a JSON object.
func pricesOf(t *testing.T, session *mcp.ClientSession) ([]byte, []map[string]any) {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: getPrices})
	if err != nil || result.IsError {
		t.Fatalf("CallTool() = %+v, %v", result, err)
	}

	var out struct {
		Companies []struct {
			Prices []map[string]any `json:"prices"`
		} `json:"companies"`
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	prices := make([]map[string]any, 0, len(out.Companies))
	for _, company := range out.Companies {
		prices = append(prices, company.Prices[0])
	}

	return raw, prices
}
