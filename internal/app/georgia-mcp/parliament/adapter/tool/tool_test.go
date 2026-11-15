package tool_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/cache"
	parliamentadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/parliament"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/tool"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
	parliamentclient "github.com/abgeo/georgia-mcp/pkg/parliament"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata        = "../../../../../../pkg/parliament/testdata/"
	searchBills     = "parliament_search_bills"
	getBill         = "parliament_get_bill"
	listVotes       = "parliament_list_votes"
	getVote         = "parliament_get_vote"
	searchQuestions = "parliament_search_questions"
	keyQuery        = "query"
	keyMember       = "member"
	keyFrom         = "from"
	keyTo           = "to"
	keyLimit        = "limit"
	keyPage         = "page"
	keyID           = "id"
	budget          = "ბიუჯეტ"
	unknownID       = 99999999
	weekStart       = "2026-11-10"
	weekEnd         = "2026-11-12"
	outage          = "The Parliament of Georgia did not answer. Try again in a few minutes."
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

// vendor is the fake Parliament: a transport that serves the recorded answers, counts the
// calls, and can fail or wait. threeVotes gives two more stages of bill 32449 a vote.
type vendor struct {
	status     int
	delay      time.Duration
	threeVotes bool
	calls      atomic.Int32
}

func (v *vendor) RoundTrip(request *http.Request) (*http.Response, error) {
	v.calls.Add(1)
	time.Sleep(v.delay)

	status, body := v.status, []byte(nil)

	if status == 0 {
		status = http.StatusOK

		read, err := os.ReadFile(testdata + file(request))
		if err != nil {
			status = http.StatusNotFound
		}

		body = read
	}

	if v.threeVotes && strings.HasSuffix(request.URL.Path, "/billReview") {
		body = bytes.Replace(body, []byte(`"voteY":null`), []byte(`"voteY":1`), 2)
	}

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    request,
	}, nil
}

// files maps a path with one answer to its recorded file.
func files() map[string]string {
	return map[string]string{
		"/law/1/bill/32449":            "bill-32449.json",
		"/law/1/bill/32449/billReview": "bill-reviews-32449.json",
		"/law/1/voting/session/31507":  "vote-session-31507.json",
		"/law/1/voting":                "votings-31507.json",
		"/mpQuestion/1/":               "questions-recent.json",
	}
}

// file picks the recorded answer of a request. An unknown identifier gives [null], and a
// text with no match gives an empty list.
func file(request *http.Request) string {
	query := request.URL.Query()

	switch path := request.URL.Path; path {
	case "/law/1/bill":
		if name := query.Get("billName"); name != "" && name != budget {
			return "empty-list.json"
		}

		return "bills-budget.json"
	case "/law/1/voteSession":
		if query.Get("billReviewId") != "" {
			return "vote-sessions-114284.json"
		}

		return "vote-sessions-week.json"
	default:
		if name, found := files()[path]; found {
			return name
		}

		return "null.json"
	}
}

// connect serves the fake Parliament through the real client, adapter, cache, and use
// cases.
func connect(t *testing.T, fake *vendor) *mcp.ClientSession {
	t.Helper()

	client := parliamentclient.New(
		vendorhttp.WithBaseURL("https://parliament.test"),
		vendorhttp.WithHTTPClient(&http.Client{Transport: fake}),
	)

	sources, err := cache.NewSourceCache(parliamentadapter.NewAdapter(client, clock))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = sources.Close() })

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, slog.New(slog.DiscardHandler))
	if err := tool.Register(registry, tool.UseCases{
		SearchBills:     app.NewSearchBills(sources),
		GetBill:         app.NewGetBill(sources),
		ListVotes:       app.NewListVotes(sources),
		GetVote:         app.NewGetVote(sources),
		SearchQuestions: app.NewSearchQuestions(sources, clock),
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

type pageResult struct {
	Source string `json:"source"`
	AsOf   string `json:"as_of"`
	Total  int    `json:"total"`
	Page   int    `json:"page"`
	Notice string `json:"notice"`
}

type bill struct {
	ID         int    `json:"id"`
	Registered string `json:"registered"`
	URL        string `json:"url"`
}

type billsResult struct {
	pageResult

	Bills []bill `json:"bills"`
}

type billResult struct {
	bill

	Stages []map[string]any `json:"stages"`
}

type vote struct {
	ID      int    `json:"id"`
	BillID  int    `json:"bill_id"`
	Reading code   `json:"reading"`
	Outcome code   `json:"outcome"`
	Yes     int    `json:"yes"`
	No      int    `json:"no"`
	Absent  int    `json:"absent"`
	URL     string `json:"url"`
}

type votesResult struct {
	pageResult

	Votes []vote `json:"votes"`
}

type faction struct {
	Name    string           `json:"name"`
	Yes     int              `json:"yes"`
	No      int              `json:"no"`
	Absent  int              `json:"absent"`
	Other   int              `json:"other"`
	Members []map[string]any `json:"members"`
}

type voteResult struct {
	vote

	Factions []faction `json:"factions"`
}

type question struct {
	ID       int    `json:"id"`
	Deadline string `json:"deadline"`
	Status   string `json:"status"`
	URL      string `json:"url"`
}

type questionsResult struct {
	pageResult

	Questions []question `json:"questions"`
}

// PARL-SC-013: A search by a Georgian text gives its source, the instant in Tbilisi, the
// total, the page, and 10 bills.
func TestSearchBills(t *testing.T) {
	t.Parallel()

	got := decode[billsResult](
		t,
		call(t, connect(t, &vendor{}), searchBills, map[string]any{keyQuery: budget}),
	)

	if got.Source != "Parliament of Georgia" || !strings.HasSuffix(got.AsOf, "+04:00") ||
		got.Total != 1088 || got.Page != 1 || len(got.Bills) != 10 ||
		got.Bills[0].Registered != "2026-11-11" || got.Notice != "" {
		t.Errorf("search = %+v, %d bills", got.pageResult, len(got.Bills))
	}
}

// PARL-SC-014: The bill, each vote of the week, and the question carry the URL of their
// page.
func TestPages(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{})

	billPage := decode[billResult](t, call(t, session, getBill, map[string]any{keyID: 32449}))
	if billPage.URL != "https://info.parliament.ge/#law-drafting/32449" ||
		len(billPage.Stages) != 17 {
		t.Errorf("bill = %s, %d stages", billPage.URL, len(billPage.Stages))
	}

	votes := decode[votesResult](t, call(t, session, listVotes, map[string]any{
		keyFrom: weekStart, keyTo: weekEnd, keyLimit: 50,
	}))
	for _, item := range votes.Votes {
		if want := "https://info.parliament.ge/#law-drafting/"; item.URL != want+strconv.Itoa(
			item.BillID,
		) {
			t.Errorf("vote %d url = %s, want the page of bill %d", item.ID, item.URL, item.BillID)
		}
	}

	questions := decode[questionsResult](
		t,
		call(t, session, searchQuestions, map[string]any{keyQuery: budget}),
	)
	if questions.Questions[0].URL != "https://info.parliament.ge/#mpqs/14583" {
		t.Errorf("question url = %s", questions.Questions[0].URL)
	}
}

// PARL-SC-015: The votes of the week hold vote 31510 with its codes, its names, and its
// counts.
func TestListVotes(t *testing.T) {
	t.Parallel()

	got := decode[votesResult](t, call(t, connect(t, &vendor{}), listVotes, map[string]any{
		keyFrom: weekStart, keyTo: weekEnd, keyLimit: 50,
	}))

	want := vote{
		ID:      31510,
		BillID:  32449,
		Reading: code{Code: "first_reading", Name: "I მოსმენა"},
		Outcome: code{Code: "adopted", Name: "საკითხი მიღებულია"},
		Yes:     77,
		Absent:  24,
		URL:     "https://info.parliament.ge/#law-drafting/32449",
	}

	if got.Total != 12 || len(got.Votes) != 12 {
		t.Fatalf("total, len = %d, %d, want 12, 12", got.Total, len(got.Votes))
	}

	for _, item := range got.Votes {
		if item.ID == want.ID && item != want {
			t.Errorf("vote = %+v, want %+v", item, want)
		}
	}
}

// PARL-SC-016: The factions of vote 31507 add up to its counts.
func TestGetVote(t *testing.T) {
	t.Parallel()

	got := decode[voteResult](
		t,
		call(t, connect(t, &vendor{}), getVote, map[string]any{keyID: 31507}),
	)

	var yes, no, absent, members int

	for _, item := range got.Factions {
		yes, no, absent = yes+item.Yes, no+item.No, absent+item.Absent
		members += len(item.Members)
	}

	sums := [5]int{got.Yes, got.No, got.Absent, yes + no + absent, members}
	if want := [5]int{79, 10, 12, 101, 101}; sums != want || yes != 79 || no != 10 {
		t.Errorf("counts, sum, members = %v, want %v", sums, want)
	}
}

// PARL-SC-017: A search by member gives question 14583, pending until 2026-11-28.
func TestSearchQuestions(t *testing.T) {
	t.Parallel()

	got := decode[questionsResult](t, call(t, connect(t, &vendor{}), searchQuestions,
		map[string]any{keyMember: "ხორგუანი"}))

	want := question{
		ID: 14583, Deadline: "2026-11-28", Status: "pending",
		URL: "https://info.parliament.ge/#mpqs/14583",
	}
	if got.Questions[0] != want {
		t.Errorf("question = %+v, want %+v", got.Questions[0], want)
	}
}

// PARL-SC-022: An English text that finds nothing gives the notice. A text with a
// Georgian letter gives none.
func TestNotice(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{})

	english := decode[billsResult](
		t,
		call(t, session, searchBills, map[string]any{keyQuery: "budget"}),
	)
	if english.Total != 0 || english.Notice != app.GeorgianOnlyNotice {
		t.Errorf("budget = total %d, notice %q", english.Total, english.Notice)
	}

	georgian := decode[billsResult](
		t,
		call(t, session, searchBills, map[string]any{keyQuery: "zzqxვ"}),
	)
	if georgian.Total != 0 || georgian.Notice != "" {
		t.Errorf("zzqxვ = total %d, notice %q, want none", georgian.Total, georgian.Notice)
	}
}

// PARL-SC-022: Each wrong input gives its message.
func TestInputErrors(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{})

	for _, tt := range inputCases() {
		if got := message(t, call(t, session, tt.tool, tt.args)); got != tt.want {
			t.Errorf("%s %v = %q, want %q", tt.tool, tt.args, got, tt.want)
		}
	}
}

type inputCase struct {
	tool string
	args map[string]any
	want string
}

func inputCases() []inputCase {
	return []inputCase{
		{searchBills, map[string]any{}, "Give at least one of query, number, from, or to."},
		{listVotes, map[string]any{}, "Give at least one of query, from, or to."},
		{searchQuestions, map[string]any{}, "Give at least one of query, member, from, or to."},
		{searchBills, map[string]any{keyQuery: "   "}, "query: use 1 to 128 characters."},
		{
			searchQuestions,
			map[string]any{keyMember: strings.Repeat("ა", 129)},
			"member: use 1 to 128 characters.",
		},
		{
			searchBills,
			map[string]any{keyFrom: "2026-02-30"},
			`from: "2026-02-30" is not a date. Use YYYY-MM-DD.`,
		},
		{
			listVotes,
			map[string]any{keyFrom: "2026-11-12", keyTo: "2026-11-10"},
			"from, to: 2026-11-12 is after 2026-11-10.",
		},
		{
			searchBills,
			map[string]any{keyQuery: budget, keyLimit: 60},
			"limit: 60 is out of range. Use 1 to 50.",
		},
		{
			searchBills,
			map[string]any{keyQuery: budget, keyPage: -1},
			"page: -1 is out of range. Use 1 or more.",
		},
		{getBill, map[string]any{keyID: 0}, "id: 0 is not an identifier. Use a positive number."},
		{
			getBill,
			map[string]any{keyID: unknownID},
			"The Parliament has no bill 99999999. Find a bill with parliament_search_bills.",
		},
		{
			getVote,
			map[string]any{keyID: unknownID},
			"The Parliament has no vote 99999999. " +
				"Find a vote with parliament_list_votes or parliament_get_bill.",
		},
	}
}

// PARL-SC-022: A Parliament that answers 500 with an empty cache gives the message of the
// outage on each tool.
func TestAFailingParliament(t *testing.T) {
	t.Parallel()

	session := connect(t, &vendor{status: http.StatusInternalServerError})

	for name, args := range map[string]map[string]any{
		searchBills:     {keyQuery: budget},
		getBill:         {keyID: 32449},
		listVotes:       {keyQuery: budget},
		getVote:         {keyID: 31507},
		searchQuestions: {keyQuery: budget},
	} {
		if got := message(t, call(t, session, name, args)); got != outage {
			t.Errorf("%s = %q, want %q", name, got, outage)
		}
	}
}

// PARL-SC-023: With 2.9 seconds for each call and three stages with a vote, the bill
// answers in less than 10 seconds of the fake clock.
func TestAnswerTime(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := &vendor{delay: 2900 * time.Millisecond, threeVotes: true}
		session := connect(t, fake)
		start := time.Now()

		got := decode[billResult](t, call(t, session, getBill, map[string]any{keyID: 32449}))

		if elapsed := time.Since(start); elapsed >= 10*time.Second {
			t.Errorf("get_bill took %v, want less than 10s", elapsed)
		}

		if calls := fake.calls.Load(); calls != 5 || len(got.Stages) != 17 {
			t.Errorf("calls, stages = %d, %d, want 5, 17", calls, len(got.Stages))
		}
	})
}
