package tool_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	dollar = "USD"

	getRates    = "rates_get_official_rates"
	listHistory = "rates_list_official_history"
)

type fakeUseCases struct {
	rates      app.OfficialRates
	history    app.OfficialHistory
	err        error
	historyErr error
}

func (f fakeUseCases) Get(context.Context, app.OfficialRatesQuery) (app.OfficialRates, error) {
	return f.rates, f.err
}

func (f fakeUseCases) List(context.Context, app.OfficialHistoryQuery) (app.OfficialHistory, error) {
	return f.history, f.historyErr
}

func connect(t *testing.T, useCases fakeUseCases) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, useCases, useCases); err != nil {
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

func call(
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

func errorText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("result = %+v, want a tool error", result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T, want text", result.Content[0])
	}

	return text.Text
}

func decode[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	if result.IsError {
		t.Fatalf("result is an error: %+v", result.Content)
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

func rateOf(
	t *testing.T,
	code domain.CurrencyCode,
	quantity int64,
	rate string,
) domain.OfficialRate {
	t.Helper()

	officialRate, err := domain.NewOfficialRate(
		code,
		string(code),
		quantity,
		decimal.RequireFromString(rate),
		decimal.Zero,
	)
	if err != nil {
		t.Fatal(err)
	}

	return officialRate
}

func dateOf(t *testing.T, value string) domain.Date {
	t.Helper()

	date, err := domain.ParseDate(value)
	if err != nil {
		t.Fatal(err)
	}

	return date
}

// NBGRATE-SC-003: A code that is not three letters, and more than 50 codes, are tool errors.
func TestGetRatesRefusesABadCode(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeUseCases{})

	tooMany := make([]any, 51)
	for i := range tooMany {
		tooMany[i] = dollar
	}

	tests := []struct {
		currencies []any
		want       string
	}{
		{currencies: []any{"US"}, want: `currencies: "US" is not a currency code.`},
		{currencies: []any{"U$D"}, want: `currencies: "U$D" is not a currency code.`},
		{currencies: tooMany, want: "currencies: 51 codes."},
	}

	for _, tt := range tests {
		got := errorText(t, call(t, session, getRates, map[string]any{"currencies": tt.currencies}))
		if !strings.HasPrefix(got, tt.want) {
			t.Errorf("error = %q, want the prefix %q", got, tt.want)
		}
	}
}

// NBGRATE-SC-008: A date after the newest set names both dates.
func TestGetRatesReportsANotPublishedDate(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeUseCases{err: &app.NotPublishedError{
		Requested: dateOf(t, "2026-10-25"),
		Newest:    dateOf(t, "2026-10-17"),
	}})

	got := errorText(t, call(t, session, getRates, map[string]any{"date": "2026-10-25"}))

	want := "The NBG has not published rates for 2026-10-25. The newest set is valid from 2026-10-17."
	if got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// NBGRATE-SC-018: A long period names its length and the limit.
func TestListHistoryReportsALongPeriod(t *testing.T) {
	t.Parallel()

	session := connect(
		t,
		fakeUseCases{historyErr: &app.PeriodTooLongError{Days: 40, Max: app.MaxHistoryDays}},
	)

	got := errorText(t, call(t, session, listHistory, map[string]any{
		"currency": dollar, "from": "2026-09-08", "to": "2026-10-17",
	}))

	if want := "to: the period has 40 days. The longest period is 31 days."; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func TestListHistoryReportsAReversedPeriod(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeUseCases{historyErr: app.ErrPeriodReversed})

	got := errorText(t, call(t, session, listHistory, map[string]any{
		"currency": dollar, "from": "2026-10-10", "to": "2026-10-01",
	}))

	if want := "to: 2026-10-01 is before from 2026-10-10."; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

type ratesOutput struct {
	ValidFrom string `json:"valid_from"`
	AsOf      string `json:"as_of"`
	Rates     []struct {
		Currency    string `json:"currency"`
		Rate        string `json:"rate"`
		RatePerUnit string `json:"rate_per_unit"`
		ValidFrom   string `json:"valid_from"`
	} `json:"rates"`
}

func setOf(t *testing.T, rates ...domain.OfficialRate) app.OfficialRates {
	t.Helper()

	published := time.Date(2026, 10, 16, 17, 1, 1, 982_000_000, domain.TbilisiZone())

	return app.OfficialRates{
		Set:    domain.NewRateSet(dateOf(t, "2026-10-17"), published, rates),
		ReadAt: published.Add(time.Hour),
	}
}

// NBGRATE-SC-027: Each rate of the output carries the valid date of its set.
func TestGetRatesGivesEachRateItsValidDate(t *testing.T) {
	t.Parallel()

	rates := setOf(
		t,
		rateOf(t, dollar, 1, "2.5982"),
		rateOf(t, "EUR", 1, "2.9274"),
		rateOf(t, "JPY", 100, "1.6455"),
	)
	out := decode[ratesOutput](t, call(t, connect(t, fakeUseCases{rates: rates}), getRates, nil))

	if len(out.Rates) != 3 {
		t.Fatalf("rates = %d, want 3", len(out.Rates))
	}

	for _, rate := range out.Rates {
		if rate.ValidFrom != "2026-10-17" {
			t.Errorf("%s valid_from = %q, want 2026-10-17", rate.Currency, rate.ValidFrom)
		}
	}

	if out.AsOf != "2026-10-16T17:01:01.982+04:00" {
		t.Errorf("as_of = %q, want 2026-10-16T17:01:01.982+04:00", out.AsOf)
	}
}

// NBGRATE-SC-029: The output keeps the digits of the NBG, trailing zeros included.
func TestGetRatesKeepsTheDigits(t *testing.T) {
	t.Parallel()

	rates := setOf(t, rateOf(t, dollar, 1, "2.6000"), rateOf(t, "AED", 10, "7.0738"))
	out := decode[ratesOutput](t, call(t, connect(t, fakeUseCases{rates: rates}), getRates, nil))

	want := map[string][2]string{dollar: {"2.6000", "2.6000"}, "AED": {"7.0738", "0.70738"}}

	for _, rate := range out.Rates {
		if got := [2]string{rate.Rate, rate.RatePerUnit}; got != want[rate.Currency] {
			t.Errorf(
				"%s rate, rate_per_unit = %v, want %v",
				rate.Currency,
				got,
				want[rate.Currency],
			)
		}
	}
}
