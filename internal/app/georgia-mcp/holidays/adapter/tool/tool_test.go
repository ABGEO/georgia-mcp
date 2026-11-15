package tool_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	nageradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/adapter/nager"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/holidays/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	nagerclient "github.com/abgeo/georgia-mcp/pkg/nager"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata     = "../../../../../../pkg/nager/testdata/"
	listHolidays = "holidays_list_holidays"
	getDay       = "holidays_get_day"
	keyLanguage  = "language"
	keyDate      = "date"
	independence = "2026-05-26"
	tuesday      = "tuesday"
	today        = "2026-11-15"
	keyYear      = "year"
	keyFrom      = "from"
	keyLimit     = "limit"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// connect serves the recorded answer of each year through the real client and adapter.
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()

	answers := map[string][]byte{}

	for _, year := range []string{"2025", "2026", "2027"} {
		name := testdata + "ge-" + year + ".json"

		body, err := os.ReadFile(name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		answers["/api/v3/PublicHolidays/"+year+"/GE"] = body
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := answers[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	source := nageradapter.NewAdapter(nagerclient.New(vendorhttp.WithBaseURL(server.URL)), clock)

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		ListHolidays: app.NewListHolidays(source, clock),
		GetDay:       app.NewGetDay(source, clock),
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

func call(
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

func decode[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()

	if result.IsError {
		t.Fatalf("CallTool() = %+v, want a result", result.Content)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var output T
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatal(err)
	}

	return output
}

type holiday struct {
	Date    string `json:"date"`
	Weekday string `json:"weekday"`
	Name    string `json:"name"`
}

type listResult struct {
	Source   string    `json:"source"`
	AsOf     string    `json:"as_of"`
	Stale    bool      `json:"stale"`
	Holidays []holiday `json:"holidays"`
}

type dayResult struct {
	Source  string `json:"source"`
	AsOf    string `json:"as_of"`
	Date    string `json:"date"`
	Weekday string `json:"weekday"`
	Holiday bool   `json:"holiday"`
	Name    string `json:"name"`
}

func find(holidays []holiday, date string) holiday {
	for _, item := range holidays {
		if item.Date == date {
			return item
		}
	}

	return holiday{}
}

// HOLIDAY-SC-013: The list of the current year names Nager.Date, carries the offset of
// Tbilisi, and gives each holiday with its weekday and its name in the language.
func TestListHolidays(t *testing.T) {
	t.Parallel()

	session := connect(t)

	english := decode[listResult](t, call(t, session, listHolidays, map[string]any{}))
	if english.Source != "Nager.Date" || !strings.HasSuffix(english.AsOf, "+04:00") ||
		len(english.Holidays) != 17 {
		t.Fatalf("list = %s %s with %d holidays, want Nager.Date, +04:00, 17",
			english.Source, english.AsOf, len(english.Holidays))
	}

	want := holiday{Date: independence, Weekday: tuesday, Name: "Independence Day"}
	if diff := cmp.Diff(want, find(english.Holidays, independence)); diff != "" {
		t.Errorf("holiday mismatch (-want +got):\n%s", diff)
	}

	georgian := decode[listResult](
		t,
		call(t, session, listHolidays, map[string]any{keyLanguage: "ka"}),
	)

	want = holiday{Date: independence, Weekday: tuesday, Name: "დამოუკიდებლობის დღე"}
	if diff := cmp.Diff(want, find(georgian.Holidays, independence)); diff != "" {
		t.Errorf("holiday in ka mismatch (-want +got):\n%s", diff)
	}
}

// HOLIDAY-SC-013: The next holidays cross into the next year.
func TestListNextHolidays(t *testing.T) {
	t.Parallel()

	list := decode[listResult](
		t,
		call(t, connect(t), listHolidays, map[string]any{keyFrom: today, keyLimit: 3}),
	)

	got := make([]string, 0, len(list.Holidays))
	for _, item := range list.Holidays {
		got = append(got, item.Date)
	}

	if diff := cmp.Diff([]string{"2026-11-23", "2027-01-01", "2027-01-02"}, got); diff != "" {
		t.Errorf("dates mismatch (-want +got):\n%s", diff)
	}
}

// HOLIDAY-SC-013: A date tells whether it is a holiday.
func TestGetDay(t *testing.T) {
	t.Parallel()

	session := connect(t)

	tests := []struct {
		args map[string]any
		want dayResult
	}{
		{
			args: map[string]any{keyDate: independence},
			want: dayResult{
				Date:    independence,
				Weekday: tuesday,
				Holiday: true,
				Name:    "Independence Day",
			},
		},
		{
			args: map[string]any{keyDate: "2026-05-27"},
			want: dayResult{Date: "2026-05-27", Weekday: "wednesday"},
		},
		{
			args: map[string]any{},
			want: dayResult{Date: today, Weekday: "sunday"},
		},
	}

	for _, test := range tests {
		got := decode[dayResult](t, call(t, session, getDay, test.args))
		if got.Source != "Nager.Date" || !strings.HasSuffix(got.AsOf, "+04:00") {
			t.Errorf("source, as_of = %s, %s, want Nager.Date and +04:00", got.Source, got.AsOf)
		}

		got.Source, got.AsOf = "", ""
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("day %v mismatch (-want +got):\n%s", test.args, diff)
		}
	}
}

type badInput struct {
	name string
	tool string
	args map[string]any
	want string
}

// badInputs holds each bad input of 4.5 with its message.
func badInputs() []badInput {
	return []badInput{
		{
			name: "language", tool: listHolidays, args: map[string]any{keyLanguage: "ru"},
			want: `language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{
			name: "limit",
			tool: listHolidays,
			args: map[string]any{keyFrom: today, keyLimit: 50},
			want: "limit: 50 is out of range. Use 1 to 20.",
		},
		{
			name: "date", tool: getDay, args: map[string]any{keyDate: "2026-02-30"},
			want: `date: "2026-02-30" is not a date. Use YYYY-MM-DD.`,
		},
		{
			name: "from", tool: listHolidays, args: map[string]any{keyFrom: "15 Nov"},
			want: `from: "15 Nov" is not a date. Use YYYY-MM-DD.`,
		},
		{
			name: "both",
			tool: listHolidays,
			args: map[string]any{keyYear: 2026, keyFrom: today},
			want: "year, from: give a year or a from date, not both.",
		},
		{
			name: "year", tool: listHolidays, args: map[string]any{keyYear: 2024},
			want: "year: 2024 is not supported. Use 2025 to 2027.",
		},
		{
			name: "day",
			tool: getDay,
			args: map[string]any{keyDate: "2028-01-01"},
			want: "date: 2028-01-01 is not in a supported year. Use a date from 2025-01-01 to 2027-12-31.",
		},
		{
			name: "from year",
			tool: listHolidays,
			args: map[string]any{keyFrom: "2028-01-01"},
			want: "from: 2028-01-01 is not in a supported year. Use a date from 2025-01-01 to 2027-12-31.",
		},
	}
}

// HOLIDAY-SC-014: Each bad input gives the message of 4.5.
func TestBadInputs(t *testing.T) {
	t.Parallel()

	session := connect(t)

	tests := badInputs()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := call(t, session, test.tool, test.args)
			if !result.IsError || len(result.Content) == 0 {
				t.Fatalf("CallTool(%s, %v) = %+v, want a tool error", test.tool, test.args, result)
			}

			text, ok := result.Content[0].(*mcp.TextContent)
			if !ok || text.Text != test.want {
				t.Errorf("message = %+v, want %q", result.Content[0], test.want)
			}
		})
	}
}

// HOLIDAY-SC-020: The description of each tool warns that a day off by decree is missing.
func TestDescriptionsNameTheDecree(t *testing.T) {
	t.Parallel()

	tools, err := connect(t).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(tools.Tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools.Tools))
	}

	for _, item := range tools.Tools {
		if !strings.Contains(item.Description, "decree") {
			t.Errorf("%s: the description names no decree", item.Name)
		}

		if item.Annotations == nil || !item.Annotations.ReadOnlyHint {
			t.Errorf("%s: not read-only", item.Name)
		}
	}
}
