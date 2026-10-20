package tool_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	searchOffers = "offers_search_offers"
	getOffer     = "offers_get_offer"
	bogName      = "Bank of Georgia"
	bogCode      = "bog"
	basis        = "basis"
	ronnyURL     = "https://bankofgeorgia.ge/en/offers-hub/details/17409"
	ronnyTerms   = "Until 22 May 2027."
	ronnyBenefit = "20% Discount, unlimited"
	ronnyTitle   = "Ronny's Pizza"
	asOf         = "2026-10-20T12:00:00+04:00"
	argBanks     = "banks"
	argBank      = "bank"
	argID        = "id"
	argQuery     = "query"
)

func bog() domain.Bank {
	return domain.Bank{Code: bogCode, Name: bogName}
}

func readAt() time.Time {
	return time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
}

// fakeUses answers fixed results, and keeps the last queries.
type fakeUses struct {
	mu        sync.Mutex
	result    app.SearchResult
	searchErr error
	detail    app.OfferDetail
	getErr    error
	lastQuery app.SearchQuery
	lastGet   app.OfferQuery
}

func (f *fakeUses) Search(_ context.Context, query app.SearchQuery) (app.SearchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.lastQuery = query

	return f.result, f.searchErr
}

func (f *fakeUses) Get(_ context.Context, query app.OfferQuery) (app.OfferDetail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.lastGet = query

	return f.detail, f.getErr
}

func (*fakeUses) Banks() []domain.Bank {
	return []domain.Bank{bog()}
}

func (*fakeUses) Cards() map[domain.BankCode][]domain.Card {
	return map[domain.BankCode][]domain.Card{
		bogCode: {{Code: "amex", Name: "Amex"}, {Code: "visa_infinite", Name: "Visa Infinite"}},
	}
}

func connect(t *testing.T, uses *fakeUses) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, uses, uses); err != nil {
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

	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T, want text", result.Content[0])
	}

	return content.Text
}

// decode reads the structured content of a successful result into out.
func decode(t *testing.T, result *mcp.CallToolResult, out any) {
	t.Helper()

	if result.IsError {
		t.Fatalf("result = %s, want a success", errorText(t, result))
	}

	body, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(body, out); err != nil {
		t.Fatal(err)
	}
}

type cardJSON struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type offerJSON struct {
	Bank           string     `json:"bank"`
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Partner        *string    `json:"partner"`
	Benefit        string     `json:"benefit"`
	Kinds          []string   `json:"kinds"`
	Categories     []string   `json:"categories"`
	BankCategories []string   `json:"bank_categories"`
	Tiers          []string   `json:"tiers"`
	Cards          []cardJSON `json:"cards"`
	StartDate      *string    `json:"start_date"`
	EndDate        *string    `json:"end_date"`
	URL            string     `json:"url"`
}

type bankJSON struct {
	Bank       string      `json:"bank"`
	Source     string      `json:"source"`
	AsOf       string      `json:"as_of"`
	Stale      bool        `json:"stale"`
	Total      int         `json:"total"`
	NextOffset *int        `json:"next_offset"`
	Offers     []offerJSON `json:"offers"`
}

type searchJSON struct {
	Meaning     string     `json:"meaning"`
	Banks       []bankJSON `json:"banks"`
	Unavailable []bankJSON `json:"unavailable"`
}

type detailJSON struct {
	Source string    `json:"source"`
	AsOf   string    `json:"as_of"`
	Stale  bool      `json:"stale"`
	Offer  offerJSON `json:"offer"`
	Terms  string    `json:"terms"`
}

func date(t *testing.T, text string) kernel.Date {
	t.Helper()

	value, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return value
}

func ronny(t *testing.T) domain.Offer {
	t.Helper()

	return domain.Offer{
		Bank:           bogCode,
		ID:             "17409",
		Title:          ronnyTitle,
		Benefit:        ronnyBenefit,
		Kinds:          []domain.BenefitKind{domain.KindDiscount},
		Categories:     []domain.Category{domain.CategoryDining},
		BankCategories: []string{"Restaurants"},
		Tiers:          []domain.Tier{domain.TierStandard, domain.TierPremium},
		Cards:          []domain.Card{{Code: "student_card", Name: "Student Card"}},
		Period:         domain.Period{Start: date(t, "2026-05-29"), End: date(t, "2027-05-22")},
		URL:            ronnyURL,
		Terms:          ronnyTerms,
	}
}

type messageCase struct {
	name string
	tool string
	err  error
	args map[string]any
	want string
}

func searchCase(name string, args map[string]any, want string) messageCase {
	return messageCase{name: name, tool: searchOffers, args: args, want: want}
}

func searchMessageCases() []messageCase {
	return []messageCase{
		searchCase("bank code", map[string]any{argBanks: []string{"BOG"}},
			`banks: "BOG" is not a bank code. Use one of: bog.`),
		searchCase("tier", map[string]any{"tiers": []string{"vip"}},
			`tiers: "vip" is not a tier. Use one of: standard, premium, private, youth.`),
		searchCase("category", map[string]any{"categories": []string{"spa"}},
			`categories: "spa" is not an offer category. Use one of: dining, leisure, travel, `+
				"health_beauty, auto, transport, shopping, electronics, home, education, kids, "+
				"entertainment, real_estate, other."),
		searchCase("kind", map[string]any{"kinds": []string{"money"}},
			`kinds: "money" is not a benefit kind. Use one of: discount, cashback, points, other.`),
		searchCase("card", map[string]any{"cards": []string{"Visa Gold"}},
			`cards: "Visa Gold" is not a card code. Use the lowercase codes of the description.`),
		searchCase("from", map[string]any{"from": "20.10.2026"},
			`from: "20.10.2026" is not a date. Use YYYY-MM-DD.`),
		searchCase("to", map[string]any{"to": "2026-13-01"},
			`to: "2026-13-01" is not a date. Use YYYY-MM-DD.`),
		searchCase("limit", map[string]any{"limit": 50}, "limit: 50 is out of range. Use 1 to 30."),
		searchCase(
			"offset",
			map[string]any{"offset": -1},
			"offset: -1 is out of range. Use 0 or more.",
		),
		searchCase(
			"blank query",
			map[string]any{argQuery: "   "},
			"query: use 1 to 100 characters.",
		),
		searchCase("long query", map[string]any{argQuery: strings.Repeat("a", 101)},
			"query: use 1 to 100 characters."),
		searchCase("language", map[string]any{"language": "ru"},
			`language: "ru" is not a language of this tool. Use ka or en.`),
	}
}

func failureCases() []messageCase {
	unknown := &app.UnknownBankError{Code: basis, Known: []domain.BankCode{bogCode}}
	unavailable := fmt.Errorf("x: %w", kernel.ErrSourceUnavailable)
	didNotAnswer := "Bank of Georgia did not answer. Try again in a few minutes."

	return []messageCase{
		{
			name: "unknown bank", tool: searchOffers, err: unknown,
			args: map[string]any{argBanks: []string{basis}},
			want: `banks: "basis" is not a bank of this server. Use one of: bog.`,
		},
		{
			name: "range", tool: searchOffers, err: fmt.Errorf("range: %w", app.ErrInvalidRange),
			args: map[string]any{}, want: "to: must be from 0 to 366 days after from.",
		},
		{
			name: "unavailable",
			tool: searchOffers,
			err:  unavailable,
			args: map[string]any{},
			want: didNotAnswer,
		},
		{
			name: "get bank code", tool: getOffer, args: map[string]any{argBank: "BOG", argID: "1"},
			want: `bank: "BOG" is not a bank code. Use one of: bog.`,
		},
		{
			name: "get unknown bank", tool: getOffer, err: unknown,
			args: map[string]any{argBank: basis, argID: "1"},
			want: `bank: "basis" is not a bank of this server. Use one of: bog.`,
		},
		{
			name: "get empty id",
			tool: getOffer,
			args: map[string]any{argBank: bogCode, argID: " "},
			want: "id: give the offer id from offers_search_offers.",
		},
		{
			name: "get not found", tool: getOffer, err: fmt.Errorf("x: %w", app.ErrOfferNotFound),
			args: map[string]any{argBank: bogCode, argID: "123"},
			want: `id: Bank of Georgia lists no current offer "123". Use offers_search_offers.`,
		},
		{
			name: "get unavailable", tool: getOffer, err: unavailable,
			args: map[string]any{argBank: bogCode, argID: "1"}, want: didNotAnswer,
		},
	}
}

// BOGOFFER-SC-022: Each invalid input and each failure gives the message of 4.5.
func TestToolsGiveTheMessages(t *testing.T) {
	t.Parallel()

	for _, tt := range append(searchMessageCases(), failureCases()...) {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session := connect(t, &fakeUses{searchErr: tt.err, getErr: tt.err})
			if got := errorText(t, call(t, session, tt.tool, tt.args)); got != tt.want {
				t.Errorf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

// BOGOFFER-SC-022: The search passes the parsed filters, and the default limit is 10.
func TestSearchPassesTheQuery(t *testing.T) {
	t.Parallel()

	uses := &fakeUses{result: app.SearchResult{Banks: []app.BankOffers{{Bank: bog()}}}}

	var out searchJSON

	decode(t, call(t, connect(t, uses), searchOffers, map[string]any{
		argBanks:     []string{bogCode},
		"tiers":      []string{"premium"},
		"categories": []string{"leisure"},
		"kinds":      []string{"cashback"},
		"cards":      []string{"solo"},
		"from":       "2026-10-01",
		"to":         "2026-10-31",
		argQuery:     "  Spa ",
		"language":   "ka",
		"offset":     5,
	}), &out)

	want := app.SearchQuery{
		Banks:      []domain.BankCode{bogCode},
		Tiers:      []domain.Tier{domain.TierPremium},
		Categories: []domain.Category{domain.CategoryLeisure},
		Kinds:      []domain.BenefitKind{domain.KindCashback},
		Cards:      []domain.CardCode{"solo"},
		From:       date(t, "2026-10-01"),
		To:         date(t, "2026-10-31"),
		Text:       "Spa",
		Language:   kernel.LanguageGeorgian,
		Limit:      10,
		Offset:     5,
	}

	if diff := cmp.Diff(want, uses.lastQuery, cmp.AllowUnexported(kernel.Date{})); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
}

func wantRonnyJSON(id string, withDates bool) offerJSON {
	out := offerJSON{
		Bank: bogCode, ID: id, Title: ronnyTitle, Benefit: ronnyBenefit,
		Kinds: []string{"discount"}, Categories: []string{"dining"},
		BankCategories: []string{"Restaurants"}, Tiers: []string{"standard", "premium"},
		URL: ronnyURL,
	}

	if withDates {
		out.Cards = []cardJSON{{Code: "student_card", Name: "Student Card"}}
		out.StartDate, out.EndDate = new("2026-05-29"), new("2027-05-22")
	}

	return out
}

// BOGOFFER-SC-022: The search gives each bank its source, as_of, total, and offers with
// the url of the page of each offer. An absent date has no key.
func TestSearchMapsTheOutput(t *testing.T) {
	t.Parallel()

	open := ronny(t)
	open.ID, open.Period, open.Cards = "1", domain.Period{}, nil

	uses := &fakeUses{result: app.SearchResult{
		Banks: []app.BankOffers{
			{
				Bank:       bog(),
				Offers:     []domain.Offer{ronny(t), open},
				Total:      12,
				NextOffset: 2,
				ReadAt:     readAt(),
			},
		},
		Unavailable: []domain.Bank{{Code: "tbc", Name: "TBC Bank"}},
	}}

	var out searchJSON

	decode(t, call(t, connect(t, uses), searchOffers, map[string]any{}), &out)

	want := searchJSON{
		Meaning: out.Meaning,
		Banks: []bankJSON{{
			Bank: bogCode, Source: bogName, AsOf: asOf, Total: 12, NextOffset: new(2),
			Offers: []offerJSON{wantRonnyJSON("17409", true), wantRonnyJSON("1", false)},
		}},
		Unavailable: []bankJSON{{Bank: "tbc", Source: "TBC Bank"}},
	}

	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	if !strings.Contains(out.Meaning, "benefit is the text of the bank") {
		t.Errorf("meaning = %q", out.Meaning)
	}
}

// BOGOFFER-SC-022: The get returns the terms and the offer with its url.
func TestGetOfferMapsTheOutput(t *testing.T) {
	t.Parallel()

	uses := &fakeUses{detail: app.OfferDetail{
		Bank: bog(), Offer: ronny(t), Terms: ronnyTerms, ReadAt: readAt(), Stale: true,
	}}

	var out detailJSON

	decode(
		t,
		call(t, connect(t, uses), getOffer, map[string]any{argBank: bogCode, argID: " 17409 "}),
		&out,
	)

	want := detailJSON{
		Source: bogName,
		AsOf:   asOf,
		Stale:  true,
		Offer:  wantRonnyJSON("17409", true),
		Terms:  ronnyTerms,
	}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	wantQuery := app.OfferQuery{Bank: bogCode, ID: "17409", Language: kernel.LanguageEnglish}
	if diff := cmp.Diff(wantQuery, uses.lastGet); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
}

// BOGOFFER-SC-022: Each tool is read-only, and each description names the card codes or
// tells the model to give the url.
func TestToolsDeclareThemselves(t *testing.T) {
	t.Parallel()

	result, err := connect(t, &fakeUses{}).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(result.Tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(result.Tools))
	}

	for _, listed := range result.Tools {
		checkReadOnly(t, listed)

		if !strings.Contains(listed.Description, "Give the person the url") {
			t.Errorf(
				"%s description lacks the url instruction: %s",
				listed.Name,
				listed.Description,
			)
		}
	}

	search := result.Tools[slices.IndexFunc(result.Tools, func(listed *mcp.Tool) bool {
		return listed.Name == searchOffers
	})]

	for _, want := range []string{"visa_infinite (Visa Infinite)", "bog (Bank of Georgia)"} {
		if !strings.Contains(search.Description, want) {
			t.Errorf("search description lacks %q: %s", want, search.Description)
		}
	}
}

func checkReadOnly(t *testing.T, listed *mcp.Tool) {
	t.Helper()

	annotations := listed.Annotations
	if annotations == nil || !annotations.ReadOnlyHint || !annotations.IdempotentHint {
		t.Errorf("%s annotations = %+v, want read-only and idempotent", listed.Name, annotations)

		return
	}

	if annotations.DestructiveHint == nil || *annotations.DestructiveHint ||
		annotations.OpenWorldHint == nil || !*annotations.OpenWorldHint {
		t.Errorf(
			"%s annotations = %+v, want not destructive and open world",
			listed.Name,
			annotations,
		)
	}
}
