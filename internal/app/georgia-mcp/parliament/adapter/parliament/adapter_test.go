package parliament_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	parliamentadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/adapter/parliament"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	parliamentclient "github.com/abgeo/georgia-mcp/pkg/parliament"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata   = "../../../../../../pkg/parliament/testdata/"
	billPath   = "/law/1/bill"
	bill32449  = "/law/1/bill/32449"
	reviews    = "/law/1/bill/32449/billReview"
	sessions   = "/law/1/voteSession"
	session    = "/law/1/voting/session/31507"
	votings    = "/law/1/voting"
	questions  = "/mpQuestion/1/"
	sittingDay = "11-11-2026"
	packageNo  = "372/2-XIმპ"
	reading1   = "I მოსმენა"
	limitKey   = "limit"
	startKey   = "start"
	budget     = "ბიუჯეტ"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC)
}

// route is the answer of one path: a file of testdata, an empty body, or a status.
type route struct {
	file    string
	status  int
	replace [2]string
}

// recorder keeps the queries that the server received, by path.
type recorder struct {
	mu      sync.Mutex
	queries map[string][]url.Values
}

func (r *recorder) of(path string) []url.Values {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.queries[path]
}

func read(t *testing.T, item route) []byte {
	t.Helper()

	if item.file == "" {
		return nil
	}

	body, err := os.ReadFile(testdata + item.file)
	if err != nil {
		t.Fatal(err)
	}

	if item.replace[0] != "" {
		body = bytes.ReplaceAll(body, []byte(item.replace[0]), []byte(item.replace[1]))
	}

	return body
}

// serve answers each path with its route, and 404 for a path with none.
func serve(t *testing.T, routes map[string]route) (*parliamentadapter.Adapter, *recorder) {
	t.Helper()

	bodies := map[string][]byte{}
	for path, item := range routes {
		bodies[path] = read(t, item)
	}

	calls := &recorder{queries: map[string][]url.Values{}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.mu.Lock()
		calls.queries[r.URL.Path] = append(calls.queries[r.URL.Path], r.URL.Query())
		calls.mu.Unlock()

		item, found := routes[r.URL.Path]
		if !found {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		w.WriteHeader(max(item.status, http.StatusOK))
		_, _ = w.Write(bodies[r.URL.Path])
	}))
	t.Cleanup(server.Close)

	client := parliamentclient.New(vendorhttp.WithBaseURL(server.URL))

	return parliamentadapter.NewAdapter(client, clock), calls
}

func date(t *testing.T, text string) *kernel.Date {
	t.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return &parsed
}

func text(t *testing.T, value string) domain.SearchText {
	t.Helper()

	parsed, err := domain.NewSearchText(value)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

func billRoutes() map[string]route {
	return map[string]route{
		bill32449: {file: "bill-32449.json"},
		reviews:   {file: "bill-reviews-32449.json"},
		sessions:  {file: "vote-sessions-114284.json"},
	}
}

// PARL-SC-007: The bill 32449 keeps its title as published, has 17 stages, and its last
// stage is the first reading with its counts and its vote. One stage has a vote, so the
// adapter sends one call of the votes.
func TestBill(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, billRoutes())

	read, err := adapter.Bill(t.Context(), 32449)
	if err != nil {
		t.Fatalf("Bill() error = %v", err)
	}

	bill := read.Bill
	bill.Description, bill.PackageName = "", ""

	want := domain.Bill{
		ID:                 32449,
		Title:              "„საქართველოს საბიუჯეტო კოდექსში ცვლილების შეტანის შესახებ“",
		Type:               "საქართველოს კანონის პროექტი",
		RegistrationNumber: packageNo,
		Initiator:          "საქართველოს მთავრობა",
		Author:             "საქართველოს ფინანსთა სამინისტრო",
		Registered:         *date(t, "2026-09-30"),
		NewestStage:        reading1,
		NewestReading:      domain.FirstReading,
		URL:                "https://info.parliament.ge/#law-drafting/32449",
	}

	if !reflect.DeepEqual(bill, want) {
		t.Errorf("bill = %+v, want %+v", bill, want)
	}

	if !read.ReadAt.Equal(clock()) || read.ReadAt.Location().String() != "Asia/Tbilisi" {
		t.Errorf("ReadAt = %v, want the clock in Tbilisi", read.ReadAt)
	}
}

// PARL-SC-007: The bill 32449 has 17 stages. The last is the first reading with its counts
// and its vote. One stage has a vote, so the adapter sends one call of the votes.
func TestBillStages(t *testing.T) {
	t.Parallel()

	adapter, calls := serve(t, billRoutes())

	read, err := adapter.Bill(t.Context(), 32449)
	if err != nil {
		t.Fatalf("Bill() error = %v", err)
	}

	if len(read.Stages) != 17 {
		t.Fatalf("len(stages) = %d, want 17", len(read.Stages))
	}

	wantStage := domain.Stage{
		Name:          reading1,
		Reading:       domain.FirstReading,
		From:          date(t, "2026-10-13"),
		To:            date(t, "2026-10-15"),
		SessionDate:   date(t, "2026-11-11"),
		SessionNumber: "82",
		BureauNumber:  "179",
		Yes:           new(77),
		No:            new(0),
		Votes:         []domain.ID{31510},
	}

	if !reflect.DeepEqual(read.Stages[16], wantStage) {
		t.Errorf("last stage = %+v, want %+v", read.Stages[16], wantStage)
	}

	if read.Stages[1].Reading != "" || read.Stages[1].Votes != nil {
		t.Errorf("committee stage = %+v, want no reading and no vote", read.Stages[1])
	}

	if got := calls.of(sessions); len(got) != 1 || got[0].Get("billReviewId") != "114284" {
		t.Errorf("vote calls = %v, want one with billReviewId 114284", got)
	}
}

// PARL-SC-008: The answer [null] gives ErrBillNotFound and ErrVoteNotFound.
func TestNotFound(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, map[string]route{
		"/law/1/bill/99":           {file: "null.json"},
		"/law/1/voting/session/99": {file: "null.json"},
	})

	if _, err := adapter.Bill(t.Context(), 99); !errors.Is(err, app.ErrBillNotFound) {
		t.Errorf("Bill() error = %v, want ErrBillNotFound", err)
	}

	if _, err := adapter.Vote(t.Context(), 99); !errors.Is(err, app.ErrVoteNotFound) {
		t.Errorf("Vote() error = %v, want ErrVoteNotFound", err)
	}
}

// PARL-SC-008: A client error, an HTML page, a date in another form, and a vote with no
// choice each give an error that wraps kernel.ErrSourceUnavailable.
func TestOutages(t *testing.T) {
	t.Parallel()

	html := route{file: "error.html"}
	badDate := [2]string{sittingDay, "2026-11-11"}

	tests := []struct {
		name   string
		routes map[string]route
		call   func(t *testing.T, adapter *parliamentadapter.Adapter) error
	}{
		{
			name: "a status of 500",
			routes: map[string]route{
				billPath: {file: "error.html", status: http.StatusInternalServerError},
			},
			call: searchBills,
		},
		{
			name:   "an HTML page of 200",
			routes: map[string]route{sessions: html},
			call:   listVotes,
		},
		{
			name:   "an empty body",
			routes: map[string]route{questions: {}},
			call:   searchQuestions,
		},
		{
			name: "a date in another form",
			routes: map[string]route{
				bill32449: {file: "bill-32449.json"},
				reviews:   {file: "bill-reviews-32449.json", replace: badDate},
				sessions:  {file: "vote-sessions-114284.json"},
			},
			call: getBill,
		},
		{
			name: "a vote with no choice",
			routes: map[string]route{
				session: {file: "vote-session-31507.json"},
				votings: {file: "empty-list.json"},
			},
			call: getVote,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := serve(t, tt.routes)

			if err := tt.call(t, adapter); !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("error = %v, want ErrSourceUnavailable", err)
			}
		})
	}
}

func searchBills(t *testing.T, adapter *parliamentadapter.Adapter) error {
	t.Helper()

	_, err := adapter.SearchBills(
		t.Context(), app.BillFilter{Number: "1"}, domain.Page{Number: 1, Size: 10},
	)

	return err
}

func listVotes(t *testing.T, adapter *parliamentadapter.Adapter) error {
	t.Helper()

	_, err := adapter.ListVotes(t.Context(), app.VoteFilter{}, domain.Page{Number: 1, Size: 10})

	return err
}

func searchQuestions(t *testing.T, adapter *parliamentadapter.Adapter) error {
	t.Helper()

	_, err := adapter.SearchQuestions(
		t.Context(), app.QuestionFilter{}, domain.Page{Number: 1, Size: 10},
	)

	return err
}

func getBill(t *testing.T, adapter *parliamentadapter.Adapter) error {
	t.Helper()

	_, err := adapter.Bill(t.Context(), 32449)

	return err
}

func getVote(t *testing.T, adapter *parliamentadapter.Adapter) error {
	t.Helper()

	_, err := adapter.Vote(t.Context(), 31507)

	return err
}

// PARL-SC-012: A bill filter turns into the dates, the number, the text, start, and limit
// of the source.
func TestSearchBillsSendsTheFilter(t *testing.T) {
	t.Parallel()

	adapter, calls := serve(t, map[string]route{billPath: {file: "bills-budget.json"}})

	period, err := domain.NewPeriod(date(t, "2026-01-01"), date(t, "2026-11-15"))
	if err != nil {
		t.Fatal(err)
	}

	page, err := adapter.SearchBills(t.Context(), app.BillFilter{
		Text: text(t, budget), Number: packageNo, Registered: period,
	}, domain.Page{Number: 3, Size: 20})
	if err != nil {
		t.Fatalf("SearchBills() error = %v", err)
	}

	want := url.Values{
		startKey: {"40"}, limitKey: {"20"}, "billName": {budget},
		"registrationNumber": {packageNo},
		"registrationDate1":  {"01-01-2026"}, "registrationDate2": {"15-11-2026"},
	}

	if got := calls.of(billPath)[0]; !reflect.DeepEqual(got, want) {
		t.Errorf("query = %v, want %v", got, want)
	}

	if page.Total != 1088 || len(page.Bills) != 10 || page.Bills[0].ID != 32559 ||
		page.Bills[0].Registered != *date(t, "2026-11-11") {
		t.Errorf("page = total %d, %d bills, first %+v", page.Total, len(page.Bills), page.Bills[0])
	}
}

// PARL-SC-012: A vote filter and a question filter turn into the query of the source.
func TestFiltersOfVotesAndQuestions(t *testing.T) {
	t.Parallel()

	adapter, calls := serve(t, map[string]route{
		sessions:  {file: "vote-sessions-week.json"},
		questions: {file: "questions-recent.json"},
	})

	period, err := domain.NewPeriod(date(t, "2026-11-10"), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, voteErr := adapter.ListVotes(t.Context(), app.VoteFilter{
		BillTitle: text(t, budget), Voted: period,
	}, domain.Page{Number: 2, Size: 10})
	_, questionErr := adapter.SearchQuestions(t.Context(), app.QuestionFilter{
		Text: text(t, budget), Member: text(t, "ხორგუანი"), Registered: period,
	}, domain.Page{Number: 1, Size: 5})

	if err := errors.Join(voteErr, questionErr); err != nil {
		t.Fatal(err)
	}

	wantVotes := url.Values{
		startKey: {"10"}, limitKey: {"10"}, "billRd": {budget}, "votingDate1": {"10-11-2026"},
	}
	if got := calls.of(sessions)[0]; !reflect.DeepEqual(got, wantVotes) {
		t.Errorf("vote query = %v, want %v", got, wantVotes)
	}

	wantQuestions := url.Values{
		startKey: {"0"}, limitKey: {"5"}, "requestTitle": {budget},
		"colleagueName": {"ხორგუანი"}, "registrationDate1": {"10-11-2026"},
	}
	if got := calls.of(questions)[0]; !reflect.DeepEqual(got, wantQuestions) {
		t.Errorf("question query = %v, want %v", got, wantQuestions)
	}
}

// PARL-SC-007: The votes of the week convert with their codes, their counts, and the page
// of their bill.
func TestListVotes(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, map[string]route{sessions: {file: "vote-sessions-week.json"}})

	period, err := domain.NewPeriod(date(t, "2026-11-10"), date(t, "2026-11-12"))
	if err != nil {
		t.Fatal(err)
	}

	page, err := adapter.ListVotes(t.Context(), app.VoteFilter{Voted: period}, domain.Page{
		Number: 1, Size: 50,
	})
	if err != nil {
		t.Fatalf("ListVotes() error = %v", err)
	}

	want := domain.Vote{
		ID:                 31510,
		Date:               *date(t, "2026-11-11"),
		BillID:             32449,
		BillTitle:          "„საქართველოს საბიუჯეტო კოდექსში ცვლილების შეტანის შესახებ“",
		RegistrationNumber: packageNo,
		Reading: domain.Coded[domain.Reading]{
			Code: domain.FirstReading,
			Name: reading1,
		},
		Outcome: domain.Coded[domain.Outcome]{
			Code: domain.Adopted,
			Name: "საკითხი მიღებულია",
		},
		Quorum: new(50),
		Yes:    77,
		Absent: 24,
		URL:    "https://info.parliament.ge/#law-drafting/32449",
	}

	if page.Total != 12 || len(page.Votes) != 12 {
		t.Fatalf("total, len = %d, %d, want 12, 12", page.Total, len(page.Votes))
	}

	for _, vote := range page.Votes {
		if vote.ID == want.ID && !reflect.DeepEqual(vote, want) {
			t.Errorf("vote = %+v, want %+v", vote, want)
		}
	}
}

// countChoices counts the choices by code, and the members of no faction.
func countChoices(choices []domain.MemberChoice) (map[domain.Choice]int, int) {
	counts := map[domain.Choice]int{}
	noFaction := 0

	for _, choice := range choices {
		counts[choice.Choice.Code]++

		if choice.Faction == "" {
			noFaction++
		}
	}

	return counts, noFaction
}

// PARL-SC-007: The vote 31507 holds its 101 choices with their codes and factions.
func TestVote(t *testing.T) {
	t.Parallel()

	adapter, calls := serve(t, map[string]route{
		session: {file: "vote-session-31507.json"},
		votings: {file: "votings-31507.json"},
	})

	read, err := adapter.Vote(t.Context(), 31507)
	if err != nil {
		t.Fatalf("Vote() error = %v", err)
	}

	got := []int{
		int(read.Vote.ID),
		read.Vote.Yes,
		read.Vote.No,
		read.Vote.Absent,
		len(read.Choices),
	}
	if want := []int{31507, 79, 10, 12, 101}; !reflect.DeepEqual(got, want) {
		t.Fatalf("id, yes, no, absent, choices = %v, want %v", got, want)
	}

	counts, noFaction := countChoices(read.Choices)

	wantCounts := map[domain.Choice]int{domain.Yes: 79, domain.No: 10, domain.Absent: 12}
	if !reflect.DeepEqual(counts, wantCounts) || noFaction != 11 {
		t.Errorf("counts = %v, no faction %d, want %v, 11", counts, noFaction, wantCounts)
	}

	if got := calls.of(votings)[0]; got.Get("voteSessionId") != "31507" ||
		got.Get(limitKey) != "200" {
		t.Errorf("votings query = %v, want vote 31507 and limit 200", got)
	}
}

// PARL-SC-014: A question has its addressee, its dates, its member, and its page.
func TestSearchQuestions(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, map[string]route{questions: {file: "questions-recent.json"}})

	page, err := adapter.SearchQuestions(t.Context(), app.QuestionFilter{}, domain.Page{
		Number: 1, Size: 3,
	})
	if err != nil {
		t.Fatalf("SearchQuestions() error = %v", err)
	}

	got := page.Questions[0]
	got.Request = ""

	want := domain.Question{
		ID:                 14583,
		Type:               "პარლამენტის წევრის შეკითხვა",
		RegistrationNumber: "6368/3-79/26",
		Registered:         *date(t, "2026-11-13"),
		Member:             "ხორგუანი სოფიო",
		Faction:            "ფრაქცია „საქართველოსთვის“",
		Addressee: "სსიპ აღსრულების ეროვნული ბიურო, მოძღვრიშვილი ლაშა, " +
			"აღსრულების ეროვნული ბიუროს თავმჯდომარეს",
		Deadline: date(t, "2026-11-28"),
		URL:      "https://info.parliament.ge/#mpqs/14583",
	}

	if !reflect.DeepEqual(got, want) || page.Total != 14183 {
		t.Errorf("question = %+v, want %+v", got, want)
	}

	wantThird := "საქართველოს მთავრობის ადმინისტრაცია, მთავრობის ადმინისტრაციას"
	if page.Questions[2].Addressee != wantThird {
		t.Errorf(
			"addressee of the third = %q, want %q, with no person",
			page.Questions[2].Addressee,
			wantThird,
		)
	}
}
