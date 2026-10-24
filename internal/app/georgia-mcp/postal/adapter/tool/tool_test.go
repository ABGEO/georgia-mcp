package tool_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	searchCodes = "postal_search_codes"
	keyQuery    = "query"
	keyLimit    = "limit"
	rustavi     = "რუსთავი"
	vendorBody  = "an unexpected error has occurred"
)

// errVendor is a failure of no known kind, with the text of a vendor body.
var errVendor = errors.New(vendorBody)

// fakeSource answers fixed addresses, or fails with err.
type fakeSource struct {
	addresses []domain.Address
	err       error
	languages []kernel.Language
}

func (f *fakeSource) Addresses(
	_ context.Context,
	_ string,
	language kernel.Language,
) (app.AddressList, error) {
	f.languages = append(f.languages, language)

	if f.err != nil {
		return app.AddressList{}, f.err
	}

	return app.AddressList{
		Addresses: f.addresses,
		ReadAt:    time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC),
	}, nil
}

type output struct {
	Source    string          `json:"source"`
	AsOf      string          `json:"as_of"`
	Stale     bool            `json:"stale"`
	Total     int             `json:"total"`
	Addresses []addressOutput `json:"addresses"`
	Note      string          `json:"note"`
}

type addressOutput struct {
	PostalCode string `json:"postal_code"`
	Address    string `json:"address"`
	District   string `json:"district"`
}

func address(t *testing.T, code, text, district string) domain.Address {
	t.Helper()

	parsed, err := domain.ParsePostalCode(code)
	if err != nil {
		t.Fatal(err)
	}

	return domain.Address{Text: text, District: district, Code: parsed}
}

// rustaviAddresses are the first three addresses of Georgian Post for რუსთავი in English.
func rustaviAddresses(t *testing.T) []domain.Address {
	t.Helper()

	return []domain.Address{
		address(t, "3700", "City Rustavi", "Rustavi City"),
		address(t, "0500", "Village Rustavi", "Aspindzis District"),
		address(t, "0114", "City Tbilisi, Rustavi highway", "Tbilisi City"),
	}
}

func connect(t *testing.T, source app.PostalCodeSource) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(
		registry,
		tool.UseCases{SearchCodes: app.NewSearchCodes(source)},
	); err != nil {
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

func callTool(t *testing.T, session *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(
		t.Context(),
		&mcp.CallToolParams{Name: searchCodes, Arguments: args},
	)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	return result
}

// search calls the tool and returns its output and its raw JSON.
func search(t *testing.T, session *mcp.ClientSession, args map[string]any) (output, string) {
	t.Helper()

	result := callTool(t, session, args)
	if result.IsError {
		t.Fatalf("CallTool() = %+v, want a result", result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out output
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out, string(raw)
}

func errorText(t *testing.T, session *mcp.ClientSession, args map[string]any) string {
	t.Helper()

	result := callTool(t, session, args)
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool() = %+v, want a tool error", result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool() content = %T, want text", result.Content[0])
	}

	return text.Text
}

// POSTCODE-SC-013: A limit out of range is refused with the range. No limit gives 20
// addresses.
func TestSearchCodesChecksTheLimit(t *testing.T) {
	t.Parallel()

	many := make([]domain.Address, 63)
	for index := range many {
		many[index] = address(
			t,
			"0179",
			fmt.Sprintf("City Tbilisi, street %d", index),
			"Tbilisi City",
		)
	}

	session := connect(t, &fakeSource{addresses: many})

	for _, limit := range []int{500, -1} {
		want := fmt.Sprintf("limit: %d is out of range. Use 1 to 100.", limit)
		if got := errorText(
			t,
			session,
			map[string]any{keyQuery: rustavi, keyLimit: limit},
		); got != want {
			t.Errorf("limit %d message = %q, want %q", limit, got, want)
		}
	}

	out, _ := search(t, session, map[string]any{keyQuery: rustavi})
	if len(out.Addresses) != 20 || out.Total != 63 {
		t.Errorf(
			"no limit = %d addresses of total %d, want 20 of 63",
			len(out.Addresses),
			out.Total,
		)
	}
}

// POSTCODE-SC-014: The output names Georgian Post, the read instant in Tbilisi, the
// total, and each address with its postal code. English is the default language.
func TestSearchCodesReturnsTheAddresses(t *testing.T) {
	t.Parallel()

	source := &fakeSource{addresses: rustaviAddresses(t)}

	out, _ := search(t, connect(t, source), map[string]any{keyQuery: rustavi})

	want := output{
		Source: "Georgian Post",
		AsOf:   "2026-10-24T12:00:00+04:00",
		Total:  3,
		Addresses: []addressOutput{
			{PostalCode: "3700", Address: "City Rustavi", District: "Rustavi City"},
			{PostalCode: "0500", Address: "Village Rustavi", District: "Aspindzis District"},
			{
				PostalCode: "0114",
				Address:    "City Tbilisi, Rustavi highway",
				District:   "Tbilisi City",
			},
		},
	}
	if diff := cmp.Diff(want, out); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}

	if len(source.languages) != 1 || source.languages[0] != kernel.LanguageEnglish {
		t.Errorf("languages = %q, want [en]", source.languages)
	}
}

// POSTCODE-SC-014: An empty list holds an empty array, a total of 0, and the note.
func TestSearchCodesExplainsAnEmptyList(t *testing.T) {
	t.Parallel()

	out, raw := search(t, connect(t, &fakeSource{}), map[string]any{keyQuery: "ზზზზზქქ"})

	if !strings.Contains(raw, `"addresses":[]`) {
		t.Errorf("output = %s, want an empty array of addresses", raw)
	}

	want := "Georgian Post found no address. In Tbilisi, Batumi, and Kutaisi, give a street. " +
		"For another town or a village, give only its name, for example Rustavi."
	if out.Total != 0 || out.Note != want {
		t.Errorf("total, note = %d, %q, want 0, %q", out.Total, out.Note, want)
	}
}

// POSTCODE-SC-014: A language other than ka and en is refused with the two codes.
// POSTCODE-SC-011: A query of fewer than 3 characters is refused with the range.
func TestSearchCodesRefusesAnInput(t *testing.T) {
	t.Parallel()

	session := connect(t, &fakeSource{})

	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "language",
			args: map[string]any{keyQuery: rustavi, "language": "ru"},
			want: `language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{
			name: "short query",
			args: map[string]any{keyQuery: "ab"},
			want: "query: give 3 to 100 characters.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := errorText(t, session, tt.args); got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
}

// POSTCODE-SC-015: A failure of Georgian Post tells the model to try again. Any other
// failure names the server, and no message holds the text of the error.
func TestSearchCodesReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "source",
			err:  fmt.Errorf("%s: %w", vendorBody, kernel.ErrSourceUnavailable),
			want: "Georgian Post did not answer. Try again in a few minutes.",
		},
		{
			name: "other",
			err:  errVendor,
			want: "The server failed to read the data of Georgian Post.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := errorText(
				t,
				connect(t, &fakeSource{err: tt.err}),
				map[string]any{keyQuery: rustavi},
			)
			if got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
}
