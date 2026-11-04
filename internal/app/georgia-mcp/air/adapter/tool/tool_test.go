package tool_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/air/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	getQuality  = "air_get_quality"
	listHistory = "air_list_history"
	keyCity     = "city"
	keyStation  = "station"
	tsrt        = "TSRT"
	tbilisiID   = 51
	tbilisi     = "Tbilisi"
	keyLanguage = "language"
	keyFrom     = "from"
	keyTo       = "to"
)

// errVendor is a failure of no known kind, with the text of a vendor body.
var errVendor = errors.New("<html>500</html>")

func clock() time.Time {
	return time.Date(2026, time.November, 5, 0, 28, 0, 0, kernel.TbilisiZone())
}

// fakeNEA answers one station of Tbilisi, or fails with err.
type fakeNEA struct {
	err error
}

func (f fakeNEA) Current(context.Context) (app.Network, error) {
	return f.network()
}

func (f fakeNEA) Recent(ctx context.Context, code domain.StationCode) (app.Network, error) {
	return f.History(ctx, code, time.Time{}, time.Time{})
}

func (f fakeNEA) History(
	_ context.Context,
	code domain.StationCode,
	_, _ time.Time,
) (app.Network, error) {
	if code.String() != tsrt {
		return app.Network{ReadAt: clock()}, f.err
	}

	return f.network()
}

func (f fakeNEA) Cities(context.Context) (app.CityList, error) {
	return app.CityList{Cities: []domain.City{
		{ID: tbilisiID, Name: domain.Name{Georgian: "თბილისი", English: tbilisi}},
	}, ReadAt: clock()}, nil
}

func (f fakeNEA) network() (app.Network, error) {
	if f.err != nil {
		return app.Network{}, f.err
	}

	code, err := domain.ParseStationCode(tsrt)
	if err != nil {
		return app.Network{}, err
	}

	at := time.Date(2026, time.November, 5, 5, 0, 0, 0, kernel.TbilisiZone())

	return app.Network{
		Stations: []domain.Station{{
			Code: code, CityID: tbilisiID,
			Address: domain.Name{
				Georgian: "ქ.თბილისი - წერეთლის გამზ., №105",
				English:  "Tbilisi - Akaki Tsereteli ave. №105",
			},
			Lat: 41.7, Lon: 44.8,
			Series: []domain.Series{{Pollutant: domain.PM10, Readings: []domain.Reading{
				{
					Value: decimal.RequireFromString(
						"51.8833333333333",
					),
					Time:  at,
					Level: domain.LevelPoor,
				},
			}}},
		}},
		ReadAt: clock(),
	}, nil
}

func connect(t *testing.T, source fakeNEA) *mcp.ClientSession {
	t.Helper()

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		GetQuality:  app.NewGetQuality(source, source, clock),
		ListHistory: app.NewListHistory(source, source, clock),
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

func errorText(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()

	result := call(t, session, name, args)
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool(%s, %v) = %+v, want a tool error", name, args, result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool() content = %T, want text", result.Content[0])
	}

	return text.Text
}

type badInput struct {
	name string
	tool string
	args map[string]any
	want string
}

// badInputs holds each bad input of 4.5 with its message.
func badInputs() []badInput {
	return append(qualityBadInputs(), historyBadInputs()...)
}

func qualityBadInputs() []badInput {
	return []badInput{
		{
			name: "city", tool: getQuality, args: map[string]any{keyCity: "Sochi"},
			want: `city: "Sochi" has no station of the NEA. Use one of: Tbilisi.`,
		},
		{
			name: "station",
			tool: getQuality,
			args: map[string]any{keyStation: "XXXX"},
			want: `station: "XXXX" is not a station of the NEA. Call air_get_quality with a city to find the codes.`,
		},
		{
			name: "language", tool: getQuality, args: map[string]any{keyLanguage: "ru"},
			want: `language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{
			name: "limit", tool: getQuality, args: map[string]any{"limit": 500},
			want: "limit: 500 is out of range. Use 1 to 100.",
		},
		{
			name: "both",
			tool: getQuality,
			args: map[string]any{keyCity: tbilisi, keyStation: tsrt},
			want: "city, station: give a city or a station, not both.",
		},
	}
}

func historyBadInputs() []badInput {
	return []badInput{
		{
			name: "malformed station",
			tool: listHistory,
			args: map[string]any{keyStation: "T"},
			want: `station: "T" is not a station of the NEA. Call air_get_quality with a city to find the codes.`,
		},
		{
			name: "unknown station",
			tool: listHistory,
			args: map[string]any{keyStation: "XXXX"},
			want: `station: "XXXX" is not a station of the NEA. Call air_get_quality with a city to find the codes.`,
		},
		{
			name: "pollutant",
			tool: listHistory,
			args: map[string]any{keyStation: tsrt, "pollutant": "PM1"},
			want: `pollutant: "PM1" is not a pollutant of the NEA. Use PM10, PM2.5, NO2, O3, SO2, or CO.`,
		},
		{
			name: "date",
			tool: listHistory,
			args: map[string]any{keyStation: tsrt, keyFrom: "5 Nov"},
			want: `from: "5 Nov" is not a date. Use YYYY-MM-DD.`,
		},
		{
			name: "period", tool: listHistory,
			args: map[string]any{keyStation: tsrt, keyFrom: "2026-10-29", keyTo: "2026-11-05"},
			want: "from, to: 2026-10-29 to 2026-11-05 is 8 days. Give at most 7.",
		},
		{
			name: "early", tool: listHistory,
			args: map[string]any{keyStation: tsrt, keyFrom: "2026-10-20", keyTo: "2026-10-22"},
			want: "from: 2026-10-20 is more than 7 days ago. Give 2026-10-29 or later.",
		},
		{
			name: "reversed", tool: listHistory,
			args: map[string]any{keyStation: tsrt, keyFrom: "2026-11-03", keyTo: "2026-11-01"},
			want: "to: 2026-11-01 is before from 2026-11-03.",
		},
		{
			name: "one date",
			tool: listHistory,
			args: map[string]any{keyStation: tsrt, keyTo: "2026-11-03"},
			want: "from, to: give both dates, or neither for the last 24 hours.",
		},
	}
}

// NEAAIR-SC-018: Each bad input gives the message of 4.5.
func TestBadInputs(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeNEA{})

	for _, tt := range badInputs() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := errorText(t, session, tt.tool, tt.args); got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
}

// NEAAIR-SC-019: A failure of the NEA, and a failure of no known kind, give a message
// with no vendor text.
func TestFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{
			err:  kernel.ErrSourceUnavailable,
			want: "The NEA did not answer. Try again in a few minutes.",
		},
		{
			err:  errVendor,
			want: "The server failed to read the data of the NEA.",
		},
	}

	for _, tt := range tests {
		session := connect(t, fakeNEA{err: tt.err})

		for _, name := range []string{getQuality, listHistory} {
			if got := errorText(
				t,
				session,
				name,
				map[string]any{keyStation: tsrt},
			); got != tt.want {
				t.Errorf("%s message = %q, want %q", name, got, tt.want)
			}
		}
	}
}

type readingJSON struct {
	Pollutant string `json:"pollutant"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
	Time      string `json:"time"`
	Level     string `json:"level"`
}

type qualityJSON struct {
	Source   string `json:"source"`
	AsOf     string `json:"as_of"`
	Total    int    `json:"total"`
	Stations []struct {
		Code     string        `json:"code"`
		Address  string        `json:"address"`
		City     string        `json:"city"`
		Level    string        `json:"level"`
		Readings []readingJSON `json:"readings"`
	} `json:"stations"`
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

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out
}

// NEAAIR-SC-017: The quality of a city names the source, writes the value as a decimal
// string and each time with +04:00, and gives the names in the language.
func TestGetQualityOutput(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeNEA{})

	out := decode[qualityJSON](t, call(t, session, getQuality, map[string]any{keyCity: tbilisi}))

	if out.Source != "National Environmental Agency" || out.AsOf != "2026-11-05T00:28:00+04:00" ||
		out.Total != 1 {
		t.Errorf("head = %q, %q, %d", out.Source, out.AsOf, out.Total)
	}

	station := out.Stations[0]
	if station.Code != tsrt || station.Address != "Tbilisi - Akaki Tsereteli ave. №105" ||
		station.City != tbilisi || station.Level != "poor" {
		t.Errorf("station = %+v", station)
	}

	want := []readingJSON{{
		Pollutant: "PM10", Value: "51.8833333333333", Unit: "µg/m³",
		Time: "2026-11-05T05:00:00+04:00", Level: "poor",
	}}
	if diff := cmp.Diff(want, station.Readings); diff != "" {
		t.Errorf("readings mismatch (-want +got):\n%s", diff)
	}

	georgian := decode[qualityJSON](t, call(t, session, getQuality,
		map[string]any{keyCity: tbilisi, keyLanguage: "ka"}))
	if georgian.Stations[0].City != "თბილისი" {
		t.Errorf("city in ka = %q", georgian.Stations[0].City)
	}
}

// NEAAIR-SC-017: The history of a station gives one series with its unit and total.
func TestListHistoryOutput(t *testing.T) {
	t.Parallel()

	session := connect(t, fakeNEA{})

	type historyJSON struct {
		Code   string `json:"code"`
		City   string `json:"city"`
		From   string `json:"from"`
		Series []struct {
			Pollutant string `json:"pollutant"`
			Unit      string `json:"unit"`
			Total     int    `json:"total"`
		} `json:"series"`
	}

	out := decode[historyJSON](t, call(t, session, listHistory, map[string]any{keyStation: "tsrt"}))

	if out.Code != tsrt || out.City != tbilisi || out.From != "2026-11-04T00:00:00+04:00" ||
		len(out.Series) != 1 || out.Series[0].Unit != "µg/m³" || out.Series[0].Total != 1 {
		t.Errorf("history = %+v", out)
	}
}
