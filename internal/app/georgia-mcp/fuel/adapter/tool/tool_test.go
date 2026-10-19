package tool_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const getPrices = "fuel_get_prices"

type fakePrices struct {
	err error
}

func (f fakePrices) Get(context.Context, app.PricesQuery) (app.Prices, error) {
	return app.Prices{}, f.err
}

func (fakePrices) Companies() []domain.Company {
	return []domain.Company{{Code: "wissol", Name: "Wissol"}}
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
