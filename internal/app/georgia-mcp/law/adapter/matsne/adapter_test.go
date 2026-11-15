package matsne_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	matsneadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/adapter/matsne"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	matsneclient "github.com/abgeo/georgia-mcp/pkg/matsne"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata  = "../../../../../../pkg/matsne/testdata/"
	taxCodeID = "1043717"
	orderID   = "5441237"
)

func clock() time.Time {
	return time.Date(2026, time.November, 15, 12, 0, 0, 0, kernel.TbilisiZone())
}

func read(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(testdata + name) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// server serves one page with a status and records the query of each request.
type server struct {
	mu      sync.Mutex
	queries []url.Values
}

func (s *server) last() url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queries[len(s.queries)-1]
}

func serve(t *testing.T, name string, status int) (*matsneadapter.Adapter, *server) {
	t.Helper()

	var body []byte
	if name != "" {
		body = read(t, name)
	}

	recorded := &server{}
	test := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.mu.Lock()
		recorded.queries = append(recorded.queries, r.URL.Query())
		recorded.mu.Unlock()

		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(test.Close)

	client := matsneclient.New(vendorhttp.WithBaseURL(test.URL))

	return matsneadapter.NewAdapter(client, clock), recorded
}

func actID(t *testing.T, text string) domain.ActID {
	t.Helper()

	id, err := domain.ParseActID(text)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func date(t *testing.T, text string) kernel.Date {
	t.Helper()

	parsed, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return parsed
}

// MATSNE-SC-009: The query maps to the parameters of the source.
func TestSearchSendsTheQuery(t *testing.T) {
	t.Parallel()

	adapter, recorded := serve(t, "search-labour-law.html", http.StatusOK)

	if _, err := adapter.Search(t.Context(), domain.SearchQuery{
		Text: "x", InText: true, Language: kernel.LanguageGeorgian,
	}); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got := recorded.last().Get("query_target"); got != "text" {
		t.Errorf("query_target = %q, want text", got)
	}

	law, active := domain.TypeLaw, domain.StatusActive
	from, to := date(t, "2010-01-01"), date(t, "2026-11-15")

	if _, err := adapter.Search(t.Context(), domain.SearchQuery{
		Text: "x", Type: &law, Status: &active, From: &from, To: &to,
		Order: domain.OldestFirst, Language: kernel.LanguageGeorgian,
	}); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	want := url.Values{
		"query": {"x"}, "query_target": {"title"}, "type": {"main"}, "group": {"1000003"},
		"status": {"active"}, "signing_date_fr[date]": {"01/01/2010"},
		"signing_date_to[date]": {"15/11/2026"}, "sort": {"signingDate_asc"},
	}
	if got := recorded.last(); got.Encode() != want.Encode() {
		t.Errorf("query = %v\nwant %v", got, want)
	}
}

// MATSNE-SC-009: The fields of an item map to a summary.
func TestSearchReadsTheSummaries(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "search-labour-law.html", http.StatusOK)

	page, err := adapter.Search(
		t.Context(),
		domain.SearchQuery{Text: "x", Language: kernel.LanguageGeorgian},
	)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	first := page.Acts[0]
	if first.Type != "საქართველოს ორგანული კანონი" || first.Issuer != "საქართველოს პარლამენტი" ||
		first.Number != "N101-IIIრს-Xმპ" || first.Signed != date(t, "2021-01-28") ||
		first.Status != domain.StatusActive || !page.ReadAt.Equal(clock()) {
		t.Errorf("first = %+v", first)
	}
}

func readAct(t *testing.T, file string, language kernel.Language) domain.Act {
	t.Helper()

	adapter, _ := serve(t, file, http.StatusOK)

	text, err := adapter.Act(t.Context(), actID(t, taxCodeID), nil, language)
	if err != nil {
		t.Fatalf("Act() error = %v", err)
	}

	return text.Act
}

func findArticle(t *testing.T, act domain.Act, number string) domain.Article {
	t.Helper()

	index := slices.IndexFunc(act.Body.Articles, func(a domain.Article) bool {
		return a.Number.String() == number
	})
	if index < 0 {
		t.Fatalf("no article %s", number)
	}

	return act.Body.Articles[index]
}

// MATSNE-SC-010: The Georgian page gives the versions and the contents.
func TestActReadsTheGeorgianVersions(t *testing.T) {
	t.Parallel()

	act := readAct(t, "act-1043717-ka.html", kernel.LanguageGeorgian)

	if act.Version != (domain.Version{Number: 248, Date: date(t, "2026-10-28")}) ||
		len(act.Versions) != 249 || act.Versions[0].Number != 248 || act.Versions[248].Number != 0 {
		t.Errorf("version = %+v of %d", act.Version, len(act.Versions))
	}

	wantContents := []domain.Heading{
		{Kind: domain.KindSection, Number: "I", Title: "ზოგადი დებულებანი"},
		{Kind: domain.KindChapter, Number: "I", Title: "საქართველოს საგადასახადო სისტემა"},
		{
			Kind:   domain.KindArticle,
			Number: "1",
			Title:  "საქართველოს საგადასახადო კოდექსის რეგულირების სფერო",
		},
	}
	if !slices.Equal(act.Body.Contents[:3], wantContents) {
		t.Errorf("contents = %+v", act.Body.Contents[:3])
	}
}

// MATSNE-SC-010: An article keeps its paragraphs and the notes of each amendment.
func TestActReadsTheGeorgianArticles(t *testing.T) {
	t.Parallel()

	act := readAct(t, "act-1043717-ka.html", kernel.LanguageGeorgian)
	article := findArticle(t, act, "133^3")

	if article.Title != "ცალკეული საქონლის მიმწოდებელი ფიზიკური პირის შემოსავლის დაბეგვრა" ||
		len(article.Paragraphs) != 5 || !strings.HasPrefix(article.Paragraphs[0], "1. ") ||
		!strings.HasPrefix(article.Paragraphs[3], "საქართველოს 2017 წლის 30 ივნისის კანონი №1182") {
		t.Errorf("133^3 = %q: %q", article.Title, article.Paragraphs)
	}
}

// MATSNE-SC-011: The English page gives the translated version and its parts.
func TestActReadsTheEnglishVersion(t *testing.T) {
	t.Parallel()

	act := readAct(t, "act-1043717-en.html", kernel.LanguageEnglish)

	if act.Title != "TAX CODE OF GEORGIA" || act.Version.Number != 247 ||
		act.Version.Date != date(t, "2026-10-15") || act.Versions[0].Number != 248 {
		t.Errorf("act = %q, version %+v, first %+v", act.Title, act.Version, act.Versions[0])
	}

	wantContents := []domain.Heading{
		{Kind: domain.KindSection, Number: "I", Title: "GENERAL PROVISIONS"},
		{Kind: domain.KindChapter, Number: "I", Title: "Georgian Tax System"},
	}
	if !slices.Equal(act.Body.Contents[:2], wantContents) {
		t.Errorf("contents = %+v", act.Body.Contents[:2])
	}
}

// MATSNE-SC-011: A bold heading opens an article, and a citation does not.
func TestActReadsTheEnglishArticles(t *testing.T) {
	t.Parallel()

	act := readAct(t, "act-1043717-en.html", kernel.LanguageEnglish)

	if article := findArticle(t, act, "133^3"); !strings.HasPrefix(article.Title,
		"Taxation of the income of a natural person") {
		t.Errorf("133^3 title = %q", article.Title)
	}

	if slices.ContainsFunc(act.Body.Articles, func(a domain.Article) bool {
		return a.Number.String() == "281"
	}) {
		t.Error("a citation of Article 281 opened an article")
	}
}

// MATSNE-SC-012: A page with no part class gives its text alone.
func TestActReadsAnActWithNoArticle(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "act-5441237-ka.html", http.StatusOK)

	text, err := adapter.Act(t.Context(), actID(t, orderID), nil, kernel.LanguageGeorgian)
	if err != nil {
		t.Fatalf("Act() error = %v", err)
	}

	body := text.Act.Body
	if len(body.Articles) != 0 || len(body.Contents) != 0 ||
		utf8.RuneCountInString(body.Text) <= 20_000 ||
		!strings.HasPrefix(body.Text, "შემოსავლების სამსახურის უფროსის") {
		t.Errorf("body = %d articles, %d headings, %d runes, want the text of more than 20,000",
			len(body.Articles), len(body.Contents), utf8.RuneCountInString(body.Text))
	}
}

// MATSNE-SC-013: An English page with no English text gives ErrNoTranslation.
func TestActChecksTheTranslation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file     string
		id       string
		language kernel.Language
		want     error
	}{
		{
			file:     "act-5441237-en.html",
			id:       orderID,
			language: kernel.LanguageEnglish,
			want:     app.ErrNoTranslation,
		},
		{file: "act-5441237-ka.html", id: orderID, language: kernel.LanguageGeorgian},
		{file: "act-1043717-en.html", id: taxCodeID, language: kernel.LanguageEnglish},
		{
			file:     "act-1043717-en-248.html",
			id:       taxCodeID,
			language: kernel.LanguageEnglish,
			want:     app.ErrNoTranslation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			t.Parallel()

			adapter, _ := serve(t, tt.file, http.StatusOK)

			_, err := adapter.Act(t.Context(), actID(t, tt.id), nil, tt.language)
			if (tt.want == nil && err != nil) || (tt.want != nil && !errors.Is(err, tt.want)) {
				t.Errorf("Act() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// MATSNE-SC-014: A refusal, a status of failure, and a broken page each tell that the
// source is unavailable.
func TestTheSourceFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		file   string
		status int
	}{
		{name: "blocked", file: "blocked.html", status: http.StatusOK},
		{name: "status 500", status: http.StatusInternalServerError},
		{name: "a broken date", file: "search-bad-date.html", status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := serve(t, tt.file, tt.status)

			_, err := adapter.Search(
				t.Context(),
				domain.SearchQuery{Text: "x", Language: kernel.LanguageGeorgian},
			)
			if !errors.Is(err, kernel.ErrSourceUnavailable) || errors.Is(err, app.ErrUnknownAct) {
				t.Errorf("Search() error = %v, want ErrSourceUnavailable alone", err)
			}
		})
	}
}

// MATSNE-SC-014: The page of the filter for an act tells both an unknown act and a refusal.
func TestABlockedActIsUnknownOrRefused(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "blocked.html", http.StatusOK)

	_, err := adapter.Act(t.Context(), actID(t, "999999999"), nil, kernel.LanguageGeorgian)
	if !errors.Is(err, app.ErrUnknownAct) || !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Act(blocked) error = %v, want ErrUnknownAct and ErrSourceUnavailable", err)
	}
}

// MATSNE-SC-014: The status comes from the class of the result of the registration code.
func TestStatusReadsTheRegistrationCode(t *testing.T) {
	t.Parallel()

	adapter, recorded := serve(t, "search-registration.html", http.StatusOK)

	result, err := adapter.Status(t.Context(), "200000000.05.001.016012")
	if err != nil || result.Status != domain.StatusActive {
		t.Errorf("Status() = %q, %v, want active", result.Status, err)
	}

	if got := recorded.last(); got.Get("registration_code") != "200000000.05.001.016012" ||
		got.Get("type") != "main" {
		t.Errorf("query = %v", got)
	}

	empty, _ := serve(t, "search-empty.html", http.StatusOK)

	result, err = empty.Status(t.Context(), "000")
	if err != nil || result.Status != "" {
		t.Errorf("Status(empty) = %q, %v, want the zero status", result.Status, err)
	}
}

// MATSNE-SC-010: A chapter in its title paragraph and an article number with a zero width
// space each open their part.
func TestActReadsTheIrregularGeorgianHeadings(t *testing.T) {
	t.Parallel()

	act := readAct(t, "act-1043717-ka.html", kernel.LanguageGeorgian)

	if !slices.Contains(act.Body.Contents, domain.Heading{
		Kind: domain.KindChapter, Number: "X", Title: "(ამოღებულია)",
	}) {
		t.Error("no chapter X (ამოღებულია)")
	}

	if article := findArticle(
		t,
		act,
		"288^3",
	); !strings.HasPrefix(
		article.Title,
		"სავალდებულო მარკირებას",
	) ||
		len(article.Paragraphs) != 1 {
		t.Errorf("288^3 = %q: %q", article.Title, article.Paragraphs)
	}
}

// MATSNE-SC-011: An English section with its title on the same line, after a soft hyphen.
func TestActReadsAnEnglishSectionWithItsTitle(t *testing.T) {
	t.Parallel()

	act := readAct(t, "act-1043717-en.html", kernel.LanguageEnglish)

	if !slices.Contains(act.Body.Contents, domain.Heading{
		Kind: domain.KindSection, Number: "X", Title: "(Deleted)",
	}) {
		t.Errorf("no section X (Deleted) in %+v", act.Body.Contents)
	}
}

// MATSNE-SC-034: An act with no list of versions has one version 0 of its publication date.
func TestActReadsAnActWithOneVersion(t *testing.T) {
	t.Parallel()

	adapter, _ := serve(t, "act-6811279-ka.html", http.StatusOK)

	text, err := adapter.Act(t.Context(), actID(t, "6811279"), nil, kernel.LanguageGeorgian)
	if err != nil {
		t.Fatalf("Act() error = %v", err)
	}

	want := domain.Version{Number: 0, Date: date(t, "2026-03-13")}

	act := text.Act
	if act.Version != want || !slices.Equal(act.Versions, []domain.Version{want}) ||
		act.Number != "38" || act.Signed != date(t, "2026-03-12") {
		t.Errorf(
			"act = number %q, signed %s, version %+v of %+v",
			act.Number,
			act.Signed,
			act.Version,
			act.Versions,
		)
	}
}
