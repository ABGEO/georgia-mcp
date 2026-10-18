package tool_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	getCommercial = "rates_get_commercial_rates"
	argCurrencies = "currencies"
	tbcCode       = "tbc"
	bogCode       = "bog"
)

type fakeCommercial struct {
	banks []domain.Bank
	err   error
}

func (f fakeCommercial) Get(
	context.Context,
	app.CommercialRatesQuery,
) (app.CommercialRates, error) {
	return app.CommercialRates{}, f.err
}

func (f fakeCommercial) Banks() []domain.Bank { return f.banks }

func connectCommercial(t *testing.T, rates fakeCommercial) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.RegisterCommercial(registry, rates); err != nil {
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

func tbcOnly() []domain.Bank {
	return []domain.Bank{{Code: tbcCode, Name: "TBC Bank"}}
}

// TBCRATE-SC-021: An unknown bank gives a message with the codes of the server.
func TestGetCommercialRatesNamesTheKnownBanks(t *testing.T) {
	t.Parallel()

	session := connectCommercial(t, fakeCommercial{
		banks: tbcOnly(),
		err:   &app.UnknownBankError{Code: bogCode, Known: []domain.BankCode{tbcCode}},
	})

	tests := map[string]string{
		"unknown":         bogCode,
		"not a bank code": "TBC",
	}

	for name, bank := range tests {
		result := call(t, session, getCommercial, map[string]any{
			argCurrencies: []string{dollar}, "banks": []string{bank},
		})

		want := fmt.Sprintf("banks: %q is not a bank of this server. Use one of: tbc.", bank)
		if got := errorText(t, result); got != want {
			t.Errorf("%s: error = %q, want %q", name, got, want)
		}
	}
}

// TBCRATE-SC-022: A wrong count of codes gives a message with the limit.
func TestGetCommercialRatesGivesTheLimitOfCodes(t *testing.T) {
	t.Parallel()

	tests := map[int]string{
		0:  "currencies: name at least one code, for example USD.",
		11: "currencies: 11 codes. Ask for 10 or fewer.",
	}

	for count, want := range tests {
		session := connectCommercial(t, fakeCommercial{
			banks: tbcOnly(),
			err:   &app.CurrencyCountError{Count: count},
		})

		result := call(t, session, getCommercial, map[string]any{argCurrencies: []string{dollar}})
		if got := errorText(t, result); got != want {
			t.Errorf("count %d: error = %q, want %q", count, got, want)
		}
	}
}

func TestGetCommercialRatesWhenEveryBankFails(t *testing.T) {
	t.Parallel()

	session := connectCommercial(t, fakeCommercial{
		banks: []domain.Bank{
			{Code: bogCode, Name: "Bank of Georgia"},
			{Code: tbcCode, Name: "TBC Bank"},
		},
		err: fmt.Errorf("asking the banks: %w", kernel.ErrSourceUnavailable),
	})

	result := call(t, session, getCommercial, map[string]any{argCurrencies: []string{dollar}})

	want := "Bank of Georgia and TBC Bank did not answer. Try again in a few minutes."
	if got := errorText(t, result); got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}
