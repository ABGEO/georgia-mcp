package parliament_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/parliament"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	budget       = "ბიუჯეტ"
	packageNo    = "372/2-XIმპ"
	sittingDay   = "11-11-2026"
	today        = "15-11-2026"
	limitKey     = "limit"
	startKey     = "start"
	adopted      = "საკითხი მიღებულია"
	firstReading = "I მოსმენა"
)

type recorded struct {
	path   string
	query  url.Values
	header http.Header
}

// serve answers each request with the file and the status, and records the request.
// An empty file name answers an empty body.
func serve(t *testing.T, file string, status int) (*parliament.Client, *[]recorded) {
	t.Helper()

	var body []byte

	if file != "" {
		read, err := os.ReadFile("testdata/" + file) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		body = read
	}

	var requests []recorded

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, recorded{
			path: r.URL.Path, query: r.URL.Query(), header: r.Header.Clone(),
		})

		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return parliament.New(vendorhttp.WithBaseURL(server.URL)), &requests
}

func fatal(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("error = %v", err)
	}
}

type requestCase struct {
	name      string
	file      string
	call      func(ctx context.Context, client *parliament.Client) error
	wantPath  string
	wantQuery url.Values
}

func ignore[T any](_ T, err error) error {
	return err
}

//nolint:funlen // One case for each method.
func requestCases() []requestCase {
	return []requestCase{
		{
			name: "bills with every filter",
			file: "bills-budget.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.Bills(ctx, parliament.BillQuery{
					Start: 40, Limit: 20, BillName: budget, RegistrationNumber: packageNo,
					RegistrationDate1: "01-01-2026", RegistrationDate2: today,
				}))
			},
			wantPath: "/law/1/bill",
			wantQuery: url.Values{
				startKey: {"40"}, limitKey: {"20"}, "billName": {budget},
				"registrationNumber": {packageNo},
				"registrationDate1":  {"01-01-2026"}, "registrationDate2": {today},
			},
		},
		{
			name: "bills with no filter",
			file: "empty-list.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.Bills(ctx, parliament.BillQuery{Limit: 10}))
			},
			wantPath:  "/law/1/bill",
			wantQuery: url.Values{startKey: {"0"}, limitKey: {"10"}},
		},
		{
			name: "bill",
			file: "bill-32449.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.Bill(ctx, 32449))
			},
			wantPath:  "/law/1/bill/32449",
			wantQuery: url.Values{},
		},
		{
			name: "bill reviews",
			file: "bill-reviews-32449.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.BillReviews(ctx, 32449))
			},
			wantPath:  "/law/1/bill/32449/billReview",
			wantQuery: url.Values{},
		},
		{
			name: "vote sessions of a period and a title",
			file: "vote-sessions-week.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.VoteSessions(ctx, parliament.VoteSessionQuery{
					Limit: 10, BillRd: budget, VotingDate1: "10-11-2026", VotingDate2: "12-11-2026",
				}))
			},
			wantPath: "/law/1/voteSession",
			wantQuery: url.Values{
				startKey: {"0"}, limitKey: {"10"}, "billRd": {budget},
				"votingDate1": {"10-11-2026"}, "votingDate2": {"12-11-2026"},
			},
		},
		{
			name: "vote sessions of a stage",
			file: "vote-sessions-114284.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.VoteSessions(ctx, parliament.VoteSessionQuery{
					Limit: 10, BillReviewID: 114284,
				}))
			},
			wantPath:  "/law/1/voteSession",
			wantQuery: url.Values{startKey: {"0"}, limitKey: {"10"}, "billReviewId": {"114284"}},
		},
		{
			name: "vote session",
			file: "vote-session-31507.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.VoteSession(ctx, 31507))
			},
			wantPath:  "/law/1/voting/session/31507",
			wantQuery: url.Values{},
		},
		{
			name: "votings",
			file: "votings-31507.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.Votings(ctx, 31507, 0, 200))
			},
			wantPath: "/law/1/voting",
			wantQuery: url.Values{
				"voteSessionId": {"31507"}, startKey: {"0"}, limitKey: {"200"}, "group": {"false"},
			},
		},
		{
			name: "member questions",
			file: "questions-recent.json",
			call: func(ctx context.Context, client *parliament.Client) error {
				return ignore(client.MPQuestions(ctx, parliament.MPQuestionQuery{
					Start: 10, Limit: 10, RequestTitle: budget, ColleagueName: "ხორგუანი",
					RegistrationDate1: "01-11-2026", RegistrationDate2: today,
				}))
			},
			wantPath: "/mpQuestion/1/",
			wantQuery: url.Values{
				startKey: {"10"}, limitKey: {"10"}, "requestTitle": {budget},
				"colleagueName":     {"ხორგუანი"},
				"registrationDate1": {"01-11-2026"}, "registrationDate2": {today},
			},
		},
	}
}

// PARL-SC-001: Each method sends its path, start and limit, the filters that the query
// sets, Accept, and the User-Agent of the server. An empty field is not sent.
func TestMethodsSendTheirRequests(t *testing.T) {
	t.Parallel()

	for _, tt := range requestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, requests := serve(t, tt.file, http.StatusOK)

			if err := tt.call(t.Context(), client); err != nil {
				t.Fatalf("error = %v", err)
			}

			got := (*requests)[0]
			if got.path != tt.wantPath {
				t.Errorf("path = %q, want %q", got.path, tt.wantPath)
			}

			if !reflect.DeepEqual(got.query, tt.wantQuery) {
				t.Errorf("query = %v, want %v", got.query, tt.wantQuery)
			}

			agent := got.header.Get("User-Agent")
			if !strings.HasPrefix(agent, "georgia-mcp/") ||
				!strings.Contains(agent, "https://github.com/abgeo/georgia-mcp") {
				t.Errorf("User-Agent = %q, want the agent of Georgia MCP", agent)
			}

			if got.header.Get("Accept") != "application/json" {
				t.Errorf("Accept = %q, want application/json", got.header.Get("Accept"))
			}
		})
	}
}

// PARL-SC-002: The recorded search gives 1088 bills, 32559 first, with its package.
func TestBillsReadsTheList(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "bills-budget.json", http.StatusOK)
	bills, err := client.Bills(t.Context(), parliament.BillQuery{Limit: 10})
	fatal(t, err)

	if bills.Total != 1088 || len(bills.List) != 10 {
		t.Fatalf("total, len = %d, %d, want 1088, 10", bills.Total, len(bills.List))
	}

	first := bills.List[0]
	got := []any{first.ID, first.BillPackage.RegistrationNumber, first.BillPackage.RegistrationDate}
	want := []any{32559, "2-15416/26/11", sittingDay}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("first bill = %v, want %v", got, want)
	}

	if len(first.BillReviewTypes) == 0 {
		t.Error("first bill has no review types")
	}
}

// PARL-SC-002: The bill 32449 has its name, its type, and its package.
func TestBill(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "bill-32449.json", http.StatusOK)
	bill, err := client.Bill(t.Context(), 32449)
	fatal(t, err)

	got := *bill
	got.BillDescription, got.BillReviewTypes, got.BillPackage.PackageName = nil, nil, ""

	want := parliament.Bill{
		ID:       32449,
		BillName: "„საქართველოს საბიუჯეტო კოდექსში ცვლილების შეტანის შესახებ“",
		BillPackage: parliament.BillPackage{
			ID:                 23124,
			RegistrationNumber: packageNo,
			Initiator:          "საქართველოს მთავრობა",
			Author:             "საქართველოს ფინანსთა სამინისტრო",
			RegistrationDate:   "30-09-2026",
			BillCount:          2,
		},
		BillType: parliament.Library{ID: 1, Name: "საქართველოს კანონის პროექტი"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("bill = %+v, want %+v", got, want)
	}

	if len(bill.BillReviewTypes) != 17 || bill.BillDescription == nil {
		t.Errorf("review types, description = %d, %v, want 17 and a text",
			len(bill.BillReviewTypes), bill.BillDescription)
	}
}

// PARL-SC-002: The bill 32449 has 17 review stages. The last has the vote counts. The
// first has none.
func TestBillReviews(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "bill-reviews-32449.json", http.StatusOK)
	reviews, err := client.BillReviews(t.Context(), 32449)
	fatal(t, err)

	if len(reviews) != 17 {
		t.Fatalf("len(reviews) = %d, want 17", len(reviews))
	}

	want := parliament.BillReview{
		ID:                 114284,
		Index:              17,
		ReviewTypeName:     firstReading,
		SessionNumber:      new("82"),
		BureauNumber:       new("179"),
		SessionReviewDate:  new(sittingDay),
		BillReviewFromDate: new("13-10-2026"),
		BillReviewToDate:   new("15-10-2026"),
		VoteY:              new(77),
		VoteN:              new(0),
	}

	if !reflect.DeepEqual(reviews[16], want) {
		t.Errorf("last review = %+v, want %+v", reviews[16], want)
	}

	if reviews[0].VoteY != nil || reviews[0].VoteN != nil {
		t.Errorf("first review has counts: %+v", reviews[0])
	}
}

// PARL-SC-002: The answer [null] gives no bill and no vote, with no error.
func TestNullGivesNothing(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "null.json", http.StatusOK)

	bill, err := client.Bill(t.Context(), 99999999)
	if err != nil || bill != nil {
		t.Errorf("Bill() = %+v, %v, want nil, nil", bill, err)
	}

	vote, err := client.VoteSession(t.Context(), 99999999)
	if err != nil || vote != nil {
		t.Errorf("VoteSession() = %+v, %v, want nil, nil", vote, err)
	}
}

// PARL-SC-002: The vote 31507 is adopted with 79 yes, 10 no, and 12 absent.
func TestVoteSession(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "vote-session-31507.json", http.StatusOK)
	vote, err := client.VoteSession(t.Context(), 31507)
	fatal(t, err)

	want := parliament.VoteSession{
		ID:                   31507,
		Quorum:               new(76),
		VotingConclusionType: &parliament.Library{ID: 1, Name: adopted},
		ImportConferenceDate: sittingDay,
		BillReviewTypeRd:     firstReading,
		BillIDRd:             32544,
		BillRd:               vote.BillRd,
		RegistrationNumberRd: "383/3-XIმპ",
		VotingResultSet:      parliament.VotingResultSet{Yes: 79, No: 10, WithoutInteraction: 12},
	}

	if !reflect.DeepEqual(*vote, want) {
		t.Errorf("vote = %+v, want %+v", *vote, want)
	}

	if !strings.Contains(vote.BillRd, "საარჩევნო კოდექსი") {
		t.Errorf("bill = %q, want the Election Code", vote.BillRd)
	}
}

// PARL-SC-002: The vote 31507 has 101 choices, 79 yes, 10 no, and 12 absent, and some
// members of no faction.
func TestVotings(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "votings-31507.json", http.StatusOK)
	votings, err := client.Votings(t.Context(), 31507, 0, 200)
	fatal(t, err)

	if votings.Total != 101 || len(votings.List) != 101 {
		t.Fatalf("total, len = %d, %d, want 101, 101", votings.Total, len(votings.List))
	}

	counts := map[string]int{}

	for _, voting := range votings.List {
		name := "no result"
		if voting.VotingResult != nil {
			name = voting.VotingResult.Name
		}

		counts[name]++

		if voting.FactionRd == nil {
			counts["no faction"]++
		}
	}

	want := map[string]int{
		"დიახ":       79,
		"არა":        10,
		"არ ესწრება": 12,
		"no faction": counts["no faction"],
	}
	if !reflect.DeepEqual(counts, want) || counts["no faction"] == 0 {
		t.Errorf("counts = %v, want %v with some members of no faction", counts, want)
	}
}

// PARL-SC-002: The week of 2026-11-10 holds 12 votes. Vote 31510 has its reading and its
// bill.
func TestVoteSessionsReadsTheList(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "vote-sessions-week.json", http.StatusOK)
	votes, err := client.VoteSessions(t.Context(), parliament.VoteSessionQuery{Limit: 50})
	fatal(t, err)

	if votes.Total != 12 || len(votes.List) != 12 {
		t.Fatalf("total, len = %d, %d, want 12, 12", votes.Total, len(votes.List))
	}

	want := parliament.VoteSession{
		ID:                   31510,
		Quorum:               new(50),
		VotingConclusionType: &parliament.Library{ID: 1, Name: adopted},
		ImportConferenceDate: sittingDay,
		BillReviewTypeRd:     firstReading,
		BillIDRd:             32449,
		BillRd:               "„საქართველოს საბიუჯეტო კოდექსში ცვლილების შეტანის შესახებ“",
		RegistrationNumberRd: packageNo,
		VotingResultSet:      parliament.VotingResultSet{Yes: 77, WithoutInteraction: 24},
	}

	for _, vote := range votes.List {
		if vote.ID == want.ID && !reflect.DeepEqual(vote, want) {
			t.Errorf("vote = %+v, want %+v", vote, want)
		}
	}
}

// PARL-SC-002: The newest question has its addressee, its deadline, and no response.
func TestMPQuestionsReadsTheList(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "questions-recent.json", http.StatusOK)
	questions, err := client.MPQuestions(t.Context(), parliament.MPQuestionQuery{Limit: 3})
	fatal(t, err)

	if questions.Total != 14183 || len(questions.List) != 3 {
		t.Fatalf("total, len = %d, %d, want 14183, 3", questions.Total, len(questions.List))
	}

	want := parliament.MPQuestion{
		ID:                 14583,
		Type:               &parliament.Library{ID: 1, Name: "პარლამენტის წევრის შეკითხვა"},
		RegistrationNumber: "6368/3-79/26",
		RegistrationDate:   "13-11-2026",
		Request: "ითხოვს ინფორმაციას სსიპ აღსრულების ეროვნულ ბიუროში მიმდინარე " +
			"გიორგი გიორგაძის სააღსრულებო საქმის თაობაზე",
		Target:             new("აღსრულების ეროვნული ბიუროს თავმჯდომარეს"),
		TargetOrganisation: new("სსიპ აღსრულების ეროვნული ბიურო"),
		TargetPerson:       new("მოძღვრიშვილი ლაშა"),
		Deadline1:          new("28-11-2026"),
		ColleagueName:      new("ხორგუანი სოფიო"),
		NodeName:           new("ფრაქცია „საქართველოსთვის“"),
	}

	if !reflect.DeepEqual(questions.List[0], want) {
		t.Errorf("first question = %+v, want %+v", questions.List[0], want)
	}
}

// PARL-SC-003: A status that is not 200 is an APIError that wraps ErrUnexpectedStatus.
func TestStatusIsAnAPIError(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, "error.html", http.StatusNotFound)

	_, err := client.Bills(t.Context(), parliament.BillQuery{Limit: 1})

	var apiError *parliament.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusNotFound {
		t.Errorf("Bills() error = %v, want an APIError with 404", err)
	}

	if !errors.Is(err, parliament.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(%v, ErrUnexpectedStatus) = false", err)
	}
}

// PARL-SC-003: An HTML page or an empty body of 200 is an error of each method.
func TestBodyThatIsNotJSON(t *testing.T) {
	t.Parallel()

	for _, file := range []string{"error.html", ""} {
		for _, tt := range requestCases() {
			t.Run(file+" "+tt.name, func(t *testing.T) {
				t.Parallel()

				client, _ := serve(t, file, http.StatusOK)

				if err := tt.call(t.Context(), client); err == nil {
					t.Error("gave no error")
				}
			})
		}
	}
}
