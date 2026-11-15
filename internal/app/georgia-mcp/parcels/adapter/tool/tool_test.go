package tool_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/cache"
	napradapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/napr"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parcels/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	naprclient "github.com/abgeo/georgia-mcp/pkg/napr"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata     = "../../../../../../pkg/napr/testdata/"
	getParcel    = "parcels_get_parcel"
	listNearby   = "parcels_list_nearby_parcels"
	search       = "parcels_search_addresses"
	freedomCode  = "01.15.05.066.020"
	chavchavadze = "ჭავჭავაძის გამზირი 44"
	keyCode      = "code"
	keyLimit     = "limit"
	keyQuery     = "query"
	keyLanguage  = "language"
	keyLat       = "lat"
	keyLon       = "lon"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// vendor is the fake NAPR: it serves the recorded answers, counts the calls, and can
// fail or wait.
type vendor struct {
	status int
	delay  time.Duration
	calls  atomic.Int32
}

func (v *vendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.calls.Add(1)
	time.Sleep(v.delay)

	if v.status != 0 {
		w.WriteHeader(v.status)

		return
	}

	body, err := os.ReadFile(testdata + v.file(r))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	_, _ = w.Write(body)
}

func (v *vendor) file(r *http.Request) string {
	switch r.URL.Path {
	case "/map/portal/search":
		body, _ := io.ReadAll(r.Body)

		switch {
		case strings.Contains(string(body), "99.99.99"):
			return "search-empty.json"
		case strings.Contains(string(body), "keyword=01."):
			return "search-code.json"
		default:
			return "search-address.json"
		}
	case "/lr/bo/mg/getinfo2":
		return "near-freedom.json"
	case "/ar/mg/getinfo":
		return "shape-address.json"
	}

	if strings.Contains(r.URL.RawQuery, "res=shp") {
		return "shape-freedom.json"
	}

	return "info-freedom.json"
}

// connect serves the fake NAPR through the real client, adapter, and cache.
func connect(t *testing.T, fake *vendor) *mcp.ClientSession {
	t.Helper()

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	adapter := napradapter.NewAdapter(naprclient.New(vendorhttp.WithBaseURL(server.URL)), clock)

	sources, err := cache.NewSourceCache(adapter)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = sources.Close() })

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		GetParcel:       app.NewGetParcel(sources),
		ListNearby:      app.NewListNearby(sources),
		SearchAddresses: app.NewSearchAddresses(sources),
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

func message(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool() = %+v, want an error", result)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Content[0] = %T, want text", result.Content[0])
	}

	return text.Text
}

type code struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type parcel struct {
	Source        string   `json:"source"`
	AsOf          string   `json:"as_of"`
	Code          string   `json:"code"`
	MapURL        string   `json:"map_url"`
	Address       string   `json:"address"`
	Area          string   `json:"area"`
	AreaUnit      string   `json:"area_unit"`
	LandType      code     `json:"land_type"`
	OwnershipType code     `json:"ownership_type"`
	Owners        []string `json:"owners"`
	Documents     []struct {
		Kind code   `json:"kind"`
		Date string `json:"date"`
		URL  string `json:"url"`
	} `json:"documents"`
	Centre struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"centre"`
}

// NAPRPARC-SC-015: A parcel carries its source, its map page, and the names of its codes
// in the language. The texts of NAPR stay Georgian.
func TestGetParcel(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{})

	english := decode[parcel](t, call(t, session, getParcel, map[string]any{keyCode: freedomCode}))

	if english.Source != "NAPR" || !strings.HasSuffix(english.AsOf, "+04:00") ||
		!strings.Contains(english.MapURL, "lbl=lr_parcels:AAArWyAATAAIs9BAAB") {
		t.Errorf(
			"source, as_of, map_url = %q, %q, %q",
			english.Source,
			english.AsOf,
			english.MapURL,
		)
	}

	want := code{Code: "non_agricultural", Name: "Non-agricultural"}
	if english.LandType != want || english.Area != "618" || english.AreaUnit != "m2" {
		t.Errorf(
			"land_type, area = %+v, %s %s, want %+v, 618 m2",
			english.LandType,
			english.Area,
			english.AreaUnit,
			want,
		)
	}

	if len(english.Documents) != 2 || english.Documents[0].Date != "2009-08-14" ||
		english.Documents[0].Kind.Name != "Registry extract" {
		t.Errorf("documents = %+v, want the extract of 2009-08-14 first", english.Documents)
	}
}

// NAPRPARC-SC-015: In ka, the codes take their Georgian names, and the texts of NAPR do
// not change.
func TestGetParcelInGeorgian(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{})

	english := decode[parcel](t, call(t, session, getParcel, map[string]any{keyCode: freedomCode}))
	georgian := decode[parcel](
		t,
		call(t, session, getParcel, map[string]any{keyCode: freedomCode, keyLanguage: "ka"}),
	)

	if georgian.LandType.Name != "არასასოფლო-სამეურნეო" || georgian.Address != english.Address {
		t.Errorf("ka land_type, address = %+v, %q", georgian.LandType, georgian.Address)
	}

	if diff := cmp.Diff(
		[]string{"ქალაქ თბილისის თვითმმართველი ერთეული"},
		georgian.Owners,
	); diff != "" {
		t.Errorf("owners (-want +got):\n%s", diff)
	}
}

// NAPRPARC-SC-016: The nearest parcels of a point come with their distance.
func TestListNearbyParcels(t *testing.T) {
	t.Parallel()

	type nearby struct {
		Parcels []struct {
			Code      string `json:"code"`
			DistanceM int    `json:"distance_m"`
		} `json:"parcels"`
	}

	got := decode[nearby](
		t,
		call(
			t,
			connect(t, &vendor{}),
			listNearby,
			map[string]any{keyLat: 41.6934, keyLon: 44.8015},
		),
	)

	if len(got.Parcels) != 5 || got.Parcels[0].Code != freedomCode ||
		got.Parcels[1].DistanceM != 92 {
		t.Errorf("parcels = %+v, want 5, Freedom Square first, 92 m second", got.Parcels)
	}
}

// NAPRPARC-SC-017: An address search in Tbilisi gives the addresses of Tbilisi with a
// centre.
func TestSearchAddresses(t *testing.T) {
	t.Parallel()

	type addresses struct {
		Addresses []struct {
			Municipality string `json:"municipality"`
			Centre       struct {
				Lat float64 `json:"lat"`
			} `json:"centre"`
		} `json:"addresses"`
	}

	got := decode[addresses](t, call(t, connect(t, &vendor{}), search, map[string]any{
		keyQuery: chavchavadze, "municipality": "თბილისი",
	}))

	if len(got.Addresses) != 5 {
		t.Fatalf("len(addresses) = %d, want 5", len(got.Addresses))
	}

	for _, address := range got.Addresses {
		if address.Municipality != "ქალაქი თბილისი" || address.Centre.Lat == 0 {
			t.Errorf("address = %+v, want Tbilisi with a centre", address)
		}
	}
}

type badInput struct {
	tool string
	args map[string]any
	want string
}

func badInputs() []badInput {
	return []badInput{
		{
			getParcel,
			map[string]any{keyCode: freedomCode, keyLanguage: "ru"},
			`language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{
			listNearby,
			map[string]any{keyLat: 41.6934, keyLon: 44.8015, keyLimit: 50},
			"limit: 50 is out of range. Use 1 to 20.",
		},
		{
			getParcel,
			map[string]any{keyCode: "01.15.05"},
			`code: "01.15.05" is not a cadastral code. Use 4 or 5 groups of digits with dots, as 01.15.05.066.020.`,
		},
		{
			getParcel,
			map[string]any{keyCode: "01.14.11.005.029.01.501"},
			`code: "01.14.11.005.029.01.501" names a unit. This tool covers parcels only. Use the parcel 01.14.11.005.029.`,
		},
		{
			listNearby,
			map[string]any{keyLat: 44.8015, keyLon: 41.6934},
			"lat, lon: 44.8015, 41.6934 is outside Georgia. " +
				"Give the latitude (41.0 to 43.6), then the longitude (40.0 to 46.8).",
		},
		{search, map[string]any{keyQuery: " "}, "query: empty. Give a street and a number."},
		{
			getParcel,
			map[string]any{keyCode: "99.99.99.999.999"},
			"code: NAPR has no parcel 99.99.99.999.999. " +
				"Find the code with parcels_list_nearby_parcels or parcels_search_addresses.",
		},
	}
}

// NAPRPARC-SC-018: Each bad input gives its message.
func TestBadInputs(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{})

	for _, input := range badInputs() {
		if got := message(t, call(t, session, input.tool, input.args)); got != input.want {
			t.Errorf("%s %v = %q, want %q", input.tool, input.args, got, input.want)
		}
	}
}

// NAPRPARC-SC-019: A failure of NAPR with an empty cache gives the message of an outage.
func TestNAPRFails(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{status: http.StatusInternalServerError})

	for _, args := range []struct {
		tool string
		args map[string]any
	}{
		{getParcel, map[string]any{keyCode: freedomCode}},
		{listNearby, map[string]any{keyLat: 41.6934, keyLon: 44.8015}},
		{search, map[string]any{keyQuery: chavchavadze}},
	} {
		got := message(t, call(t, session, args.tool, args.args))
		if got != "NAPR did not answer. Try again in a few minutes." {
			t.Errorf("%s = %q, want the outage message", args.tool, got)
		}
	}
}

// NAPRPARC-SC-022: A parcel costs 3 calls, a point 1, and a search of 3 addresses 4. The
// same calls again cost none.
func TestCallCounts(t *testing.T) {
	t.Parallel()

	fake := &vendor{}
	session := connect(t, fake)

	steps := []struct {
		tool string
		args map[string]any
		want int32
	}{
		{getParcel, map[string]any{keyCode: freedomCode}, 3},
		{listNearby, map[string]any{keyLat: 41.6934, keyLon: 44.8015}, 1},
		{search, map[string]any{keyQuery: chavchavadze, keyLimit: 3}, 4},
	}

	for _, again := range []bool{false, true} {
		for _, step := range steps {
			before := fake.calls.Load()

			decode[map[string]any](t, call(t, session, step.tool, step.args))

			want := step.want
			if again {
				want = 0
			}

			if got := fake.calls.Load() - before; got != want {
				t.Errorf("%s (again %t) sent %d calls, want %d", step.tool, again, got, want)
			}
		}
	}
}

// NAPRPARC-SC-024: With 500 ms for each call, a search of 20 addresses answers in less
// than 10 seconds and a parcel in less than 2 seconds.
func TestAnswerTimes(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{delay: 500 * time.Millisecond})

	start := time.Now()

	decode[map[string]any](
		t,
		call(t, session, search, map[string]any{keyQuery: chavchavadze, keyLimit: 20}),
	)

	if elapsed := time.Since(start); elapsed >= 10*time.Second {
		t.Errorf("search took %v, want less than 10s", elapsed)
	}

	start = time.Now()

	decode[map[string]any](t, call(t, session, getParcel, map[string]any{keyCode: freedomCode}))

	if elapsed := time.Since(start); elapsed >= 2*time.Second {
		t.Errorf("parcel took %v, want less than 2s", elapsed)
	}
}
