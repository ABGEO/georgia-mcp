package law_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	testdata     = "../../../../pkg/matsne/testdata/"
	searchActs   = "law_search_acts"
	getAct       = "law_get_act"
	taxCodeID    = "1043717"
	taxCodeTitle = "საქართველოს საგადასახადო კოდექსი"
	orderID      = "5441237"
	keyID        = "id"
	keyQuery     = "query"
	keyLang      = "language"
	keyArticle   = "articles"
	outage       = "The Legislative Herald did not answer. Try again in a few minutes."
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// Modes of the fake vendor.
const (
	modeServe int32 = iota
	modeFail
	modeBlock
)

// fakeHerald serves the recorded pages by path, or fails as its mode says.
type fakeHerald struct {
	server *httptest.Server
	mode   atomic.Int32
}

func newFakeHerald(t *testing.T) *fakeHerald {
	t.Helper()

	read := func(name string) []byte {
		body, err := os.ReadFile(testdata + name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		return body
	}

	pages := map[string][]byte{}
	for _, name := range []string{
		"search-labour-law.html", "search-registration.html", "search-en.html",
		"suggest-tax-code.json", "act-1043717-ka.html", "act-1043717-en.html",
		"act-1043717-en-248.html", "act-5441237-ka.html", "act-5441237-en.html", "blocked.html",
		"act-6811279-ka.html",
	} {
		pages[name] = read(name)
	}

	fake := &fakeHerald{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch fake.mode.Load() {
		case modeFail:
			w.WriteHeader(http.StatusInternalServerError)

			return
		case modeBlock:
			_, _ = w.Write(pages["blocked.html"])

			return
		}

		_, _ = w.Write(pages[pageOf(r)])
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

// pageOf returns the recorded page of a request. Any other path gives the page of the
// filter, as the source does for an unknown act.
func pageOf(r *http.Request) string {
	query := r.URL.Query()

	switch {
	case r.URL.Path == "/ka/document/search" && query.Get("registration_code") != "":
		return "search-registration.html"
	case r.URL.Path == "/en/document/view/"+taxCodeID && query.Get("publication") == "248":
		return "act-1043717-en-248.html"
	}

	page, ok := map[string]string{
		"/ka/document/search":            "search-labour-law.html",
		"/en/document/search":            "search-en.html",
		"/document/suggest":              "suggest-tax-code.json",
		"/ka/document/view/" + taxCodeID: "act-1043717-ka.html",
		"/en/document/view/" + taxCodeID: "act-1043717-en.html",
		"/ka/document/view/" + orderID:   "act-5441237-ka.html",
		"/en/document/view/" + orderID:   "act-5441237-en.html",
		"/ka/document/view/6811279":      "act-6811279-ka.html",
	}[r.URL.Path]
	if !ok {
		return "blocked.html"
	}

	return page
}

func (f *fakeHerald) config(t *testing.T) law.Config {
	t.Helper()

	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return law.Config{Matsne: vendors.Config{
		Enabled: true,
		BaseURL: *base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 10},
	}}
}

func serve(t *testing.T, cfg law.Config) *mcp.ClientSession {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)

	module, err := law.NewWithClock(cfg, logger, clock)
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

func decode[T any](t *testing.T, session *mcp.ClientSession, name string, args map[string]any) T {
	t.Helper()

	result := call(t, session, name, args)
	if result.IsError {
		t.Fatalf("%s(%v) = %+v, want a result", name, args, result.Content[0])
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

func errorText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	text, ok := result.Content[0].(*mcp.TextContent)
	if !result.IsError || !ok {
		t.Fatalf("result = %+v, want a tool error", result.Content)
	}

	return text.Text
}

type searchOut struct {
	Source string `json:"source"`
	AsOf   string `json:"as_of"`
	Total  int    `json:"total"`
	Page   int    `json:"page"`
	Acts   []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"acts"`
}

type versionOut struct {
	Number int    `json:"number"`
	Date   string `json:"date"`
}

type actOut struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Number   string       `json:"number"`
	Signed   string       `json:"signed"`
	Status   string       `json:"status"`
	URL      string       `json:"url"`
	Official bool         `json:"official"`
	Notice   string       `json:"notice"`
	Version  versionOut   `json:"version"`
	Newest   *versionOut  `json:"newest_version"`
	Versions []versionOut `json:"versions"`
	Count    int          `json:"version_count"`
	Contents []struct {
		Kind   string `json:"kind"`
		Number string `json:"number"`
	} `json:"contents"`
	Articles []struct {
		Number     string   `json:"number"`
		Paragraphs []string `json:"paragraphs"`
	} `json:"articles"`
}

// readOnly reports whether the annotations are those of MCP-004.
func readOnly(annotations *mcp.ToolAnnotations) bool {
	return annotations != nil && annotations.ReadOnlyHint && annotations.IdempotentHint &&
		annotations.DestructiveHint != nil && !*annotations.DestructiveHint &&
		annotations.OpenWorldHint != nil && *annotations.OpenWorldHint
}

// MATSNE-SC-031: The switch decides the two read-only tools.
func TestTheSwitchDecidesTheTools(t *testing.T) {
	t.Parallel()

	on, err := serve(t, newFakeHerald(t).config(t)).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(on.Tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(on.Tools))
	}

	for _, item := range on.Tools {
		if (item.Name != searchActs && item.Name != getAct) || !readOnly(item.Annotations) {
			t.Errorf("tool %s, want a read-only tool of law", item.Name)
		}
	}

	off, err := serve(t, law.Config{}).ListTools(t.Context(), nil)
	if err != nil || len(off.Tools) != 0 {
		t.Errorf("tools with the switch off = %d, %v, want none", len(off.Tools), err)
	}
}

// MATSNE-SC-025: A search gives the source, the instant, the page, and the address of
// each act.
func TestSearchActsReadsTheHerald(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeHerald(t).config(t))

	out := decode[searchOut](t, session, searchActs, map[string]any{
		keyQuery: "შრომის კოდექსი", "type": "law", keyLang: "ka",
	})

	if out.Source != "Legislative Herald of Georgia" || out.AsOf != "2026-11-15T12:00:00+04:00" ||
		out.Page != 1 || out.Total != 3 || len(out.Acts) != 3 {
		t.Fatalf(
			"head = %q, %q, page %d, total %d of %d",
			out.Source,
			out.AsOf,
			out.Page,
			out.Total,
			len(out.Acts),
		)
	}

	for _, act := range out.Acts {
		if act.URL != "https://matsne.gov.ge/ka/document/view/"+act.ID {
			t.Errorf("url = %q for %s", act.URL, act.ID)
		}
	}
}

// MATSNE-SC-026: The overview of the Tax Code in Georgian gives its metadata.
func TestGetActGivesTheMetadata(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeHerald(t).config(t))

	out := decode[actOut](t, session, getAct, map[string]any{keyID: taxCodeID, keyLang: "ka"})

	if out.ID != taxCodeID || out.Title != taxCodeTitle || out.Number != "3591" ||
		out.Signed != "2010-09-17" || out.Status != "active" || !out.Official || out.Notice != "" {
		t.Errorf("metadata = %+v", out)
	}
}

// MATSNE-SC-026: The overview gives the versions, the contents, and the address.
func TestGetActGivesTheOverview(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeHerald(t).config(t))

	out := decode[actOut](t, session, getAct, map[string]any{keyID: taxCodeID, keyLang: "ka"})

	if out.Version != (versionOut{Number: 248, Date: "2026-10-28"}) || len(out.Versions) != 20 ||
		out.Count != 249 ||
		out.URL != "https://matsne.gov.ge/ka/document/view/1043717?publication=248" {
		t.Errorf(
			"version = %+v of %d of %d, url %q",
			out.Version,
			len(out.Versions),
			out.Count,
			out.URL,
		)
	}

	if len(out.Contents) == 0 || out.Contents[0].Kind != "section" ||
		out.Contents[0].Number != "I" {
		t.Errorf("contents = %+v, want section I first", out.Contents)
	}
}

func englishArticle(t *testing.T) actOut {
	t.Helper()

	session := serve(t, newFakeHerald(t).config(t))

	return decode[actOut](t, session, getAct, map[string]any{
		keyID: taxCodeID, keyArticle: []string{"133^3"},
	})
}

// MATSNE-SC-026: An act that was never consolidated has one version 0 of its
// publication date.
func TestGetActGivesTheOneVersion(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeHerald(t).config(t))

	out := decode[actOut](t, session, getAct, map[string]any{keyID: "6811279", keyLang: "ka"})

	version := versionOut{Number: 0, Date: "2026-03-13"}
	if out.Version != version || len(out.Versions) != 1 || out.Versions[0] != version ||
		out.Count != 1 ||
		out.URL != "https://matsne.gov.ge/ka/document/view/6811279?publication=0" {
		t.Errorf("version = %+v of %+v, url %q", out.Version, out.Versions, out.URL)
	}
}

// MATSNE-SC-027: An article in English holds the paragraphs of the page.
func TestGetActGivesAnEnglishArticle(t *testing.T) {
	t.Parallel()

	out := englishArticle(t)

	if len(out.Articles) != 1 || out.Articles[0].Number != "133^3" ||
		len(out.Articles[0].Paragraphs) != 5 ||
		!strings.HasPrefix(out.Articles[0].Paragraphs[0], "1. The Minister of Finance of Georgia") {
		t.Errorf("articles = %+v", out.Articles)
	}
}

// MATSNE-SC-027: An English text that trails names the newer Georgian version.
func TestGetActNamesTheNewestVersion(t *testing.T) {
	t.Parallel()

	out := englishArticle(t)
	wantNotice := "This is an English translation. Only the Georgian text is official. " +
		"The newest version is 248 of 2026-10-28. Ask for language ka to read it."

	if out.Version.Number != 247 || out.Newest == nil ||
		*out.Newest != (versionOut{Number: 248, Date: "2026-10-28"}) ||
		out.Official || out.Notice != wantNotice || len(out.Versions) != 0 {
		t.Errorf("version %+v, newest %+v, official %v, notice %q", out.Version, out.Newest,
			out.Official, out.Notice)
	}
}

type errorCase struct {
	tool string
	args map[string]any
	want string
}

func searchErrors() []errorCase {
	types := "law, parliament_resolution, presidential_decree, government_resolution, " +
		"government_decree, edict, order, international_treaty, constitutional_court_decision, " +
		"court_act, self_government_ordinance, self_government_decision, revenue_service_order, " +
		"individual_legal_act, eu_law, draft"

	return []errorCase{
		{
			searchActs,
			map[string]any{keyQuery: "x", keyLang: "ru"},
			`language: "ru" is not a language of this tool. Use ka or en.`,
		},
		{searchActs, map[string]any{keyQuery: " "}, "query: give 1 to 128 characters."},
		{
			searchActs,
			map[string]any{keyQuery: strings.Repeat("ა", 129)},
			"query: give 1 to 128 characters.",
		},
		{
			searchActs,
			map[string]any{keyQuery: "x", "target": "body"},
			`target: "body" is not a target. Use title or text.`,
		},
		{
			searchActs,
			map[string]any{keyQuery: "x", "type": "statute"},
			`type: "statute" is not a document type. Use one of: ` + types + ".",
		},
		{
			searchActs,
			map[string]any{keyQuery: "x", "status": "valid"},
			`status: "valid" is not a status. Use active, expired, or pending.`,
		},
		{
			searchActs,
			map[string]any{keyQuery: "x", "signed_from": "5 Nov"},
			`signed_from: "5 Nov" is not a date. Use YYYY-MM-DD.`,
		},
		{
			searchActs,
			map[string]any{keyQuery: "x", "limit": 50},
			"limit: 50 is out of range. Use 1 to 20.",
		},
		{
			searchActs,
			map[string]any{keyQuery: "x", "page": -1},
			"page: -1 is out of range. Use 1 or more.",
		},
		{
			searchActs,
			map[string]any{keyQuery: "x", "order": "x"},
			`order: "x" is not an order. Use newest or oldest.`,
		},
	}
}

func actErrors() []errorCase {
	twelve := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}

	return []errorCase{
		{
			getAct,
			map[string]any{keyID: "abc"},
			`id: "abc" is not an act identifier. Call law_search_acts to find the identifier.`,
		},
		{
			getAct,
			map[string]any{keyID: taxCodeID, keyArticle: []string{"82a"}},
			`articles: "82a" is not an article number. Use digits, with ^ and digits for an ` +
				`inserted article, for example 133^3.`,
		},
		{
			getAct,
			map[string]any{keyID: taxCodeID, keyArticle: twelve},
			"articles: 12 articles. Give at most 10.",
		},
		{
			getAct,
			map[string]any{keyID: taxCodeID, "version": 1, "date": "2020-01-01"},
			"version, date: give a version or a date, not both.",
		},
		{
			getAct,
			map[string]any{keyID: taxCodeID, "version": 300},
			"version: the act has versions 0 (2010-10-12) to 248 (2026-10-28).",
		},
		{
			getAct,
			map[string]any{keyID: orderID, keyLang: "ka", "piece": 3},
			"piece: the text has 1 piece.",
		},
		{
			getAct,
			map[string]any{keyID: orderID, keyLang: "ka", "piece": -1},
			"piece: -1 is out of range. Use 1 or more.",
		},
		{
			getAct,
			map[string]any{keyID: orderID, keyLang: "ka", keyArticle: []string{"1"}},
			"articles: the act has no article 1. It has no articles: ask with no article to read its text.",
		},
		{
			getAct,
			map[string]any{keyID: orderID},
			"The act has no English translation of this version at the Legislative Herald. Ask for language ka.",
		},
		{
			getAct,
			map[string]any{keyID: "999999999"},
			"id: the Legislative Herald returned no act 999999999. Check the identifier with " +
				"law_search_acts, or try again in a few minutes.",
		},
	}
}

// MATSNE-SC-028: Each wrong input gives its message.
func TestEachErrorGivesItsMessage(t *testing.T) {
	t.Parallel()

	session := serve(t, newFakeHerald(t).config(t))

	for _, tt := range append(searchErrors(), actErrors()...) {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := errorText(t, call(t, session, tt.tool, tt.args)); got != tt.want {
				t.Errorf("%s(%v) = %q\nwant %q", tt.tool, tt.args, got, tt.want)
			}
		})
	}

	got := errorText(t, call(t, session, getAct, map[string]any{
		keyID: taxCodeID, keyArticle: []string{"999"},
	}))
	if !strings.HasPrefix(
		got,
		"articles: the act has no article 999. Its articles run from 1 to ",
	) {
		t.Errorf("article 999 = %q", got)
	}
}

// MATSNE-SC-029: A failing source with an empty cache gives the message of the outage.
func TestAFailingHerald(t *testing.T) {
	t.Parallel()

	fake := newFakeHerald(t)
	session := serve(t, fake.config(t))

	for i, mode := range []int32{modeFail, modeBlock} {
		fake.mode.Store(mode)

		args := map[string]any{keyQuery: "კოდექსი " + strings.Repeat("ა", i+1)}
		if got := errorText(t, call(t, session, searchActs, args)); got != outage {
			t.Errorf("mode %d: %q, want %q", mode, got, outage)
		}
	}
}
