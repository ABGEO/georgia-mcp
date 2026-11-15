package matsne_test

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/matsne"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func read(t *testing.T, name string) []byte {
	t.Helper()

	body, err := fs.ReadFile(os.DirFS("testdata"), name)
	if err != nil {
		t.Fatal(err)
	}

	return body
}

func serve(t *testing.T, body []byte, status int) *matsne.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return matsne.New(vendorhttp.WithBaseURL(server.URL))
}

type recorded struct {
	method, path, query, body, contentType string
}

// recorder serves a page for each path, and records each request after it checks the
// headers that every request carries.
func recorder(t *testing.T) (*matsne.Client, func() []recorded) {
	t.Helper()

	var (
		mu  sync.Mutex
		got []recorded
	)

	search, act := read(t, "search-empty.html"), read(t, "act-5441237-ka.html")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkHeaders(t, r)

		body, _ := io.ReadAll(r.Body)
		call := recorded{
			method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: string(body),
			contentType: r.Header.Get("Content-Type"),
		}

		func() {
			mu.Lock()
			defer mu.Unlock()

			got = append(got, call)
		}()

		switch {
		case strings.HasSuffix(r.URL.Path, "/suggest"):
			_, _ = w.Write([]byte(`[]`))
		case strings.HasSuffix(r.URL.Path, "/search"):
			_, _ = w.Write(search)
		default:
			_, _ = w.Write(act)
		}
	}))
	t.Cleanup(server.Close)

	return matsne.New(vendorhttp.WithBaseURL(server.URL)), func() []recorded {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(got)
	}
}

func checkHeaders(t *testing.T, r *http.Request) {
	t.Helper()

	agent := r.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}

	if accept := r.Header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}
}

// MATSNE-SC-001: Each method sends its path, its query or its form, the Accept header,
// and the User-Agent.
func TestClientSendsTheRequests(t *testing.T) {
	t.Parallel()

	client, calls := recorder(t)
	ctx := t.Context()

	query := matsne.SearchQuery{Query: "კოდექსი", Type: "main", Page: 2}
	if _, err := client.Search(ctx, "ka", query); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if _, err := client.Suggest(ctx, "საგადასახადო კოდექსი"); err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}

	if _, err := client.Document(ctx, "ka", "5441237", 200); err != nil {
		t.Fatalf("Document(200) error = %v", err)
	}

	if _, err := client.Document(ctx, "en", "5441237", -1); err != nil {
		t.Fatalf("Document(-1) error = %v", err)
	}

	want := []recorded{
		{
			method: http.MethodGet, path: "/ka/document/search",
			query: url.Values{"query": {"კოდექსი"}, "type": {"main"}, "page": {"1"}}.Encode(),
		},
		{
			method: http.MethodPost, path: "/document/suggest",
			body:        url.Values{"text": {"საგადასახადო კოდექსი"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
		},
		{method: http.MethodGet, path: "/ka/document/view/5441237", query: "publication=200"},
		{method: http.MethodGet, path: "/en/document/view/5441237"},
	}

	if got := calls(); !slices.Equal(got, want) {
		t.Errorf("requests = %+v\nwant %+v", got, want)
	}
}

// MATSNE-SC-002: The client reads the total and each item of a recorded search page.
func TestSearchReadsTheRecordedPages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file      string
		wantTotal string
		wantItems int
	}{
		{file: "search-tax-code.html", wantTotal: "29", wantItems: 20},
		{file: "search-en.html", wantTotal: "1", wantItems: 1},
		{file: "search-empty.html", wantTotal: "0", wantItems: 0},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			t.Parallel()

			page, err := serve(t, read(t, tt.file), http.StatusOK).
				Search(t.Context(), "ka", matsne.SearchQuery{Query: "x"})
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if page.Total != tt.wantTotal || len(page.Items) != tt.wantItems {
				t.Fatalf("Total = %q, items = %d, want %q and %d",
					page.Total, len(page.Items), tt.wantTotal, tt.wantItems)
			}

			for _, item := range page.Items {
				checkItem(t, item)
			}
		})
	}
}

func checkItem(t *testing.T, item matsne.SearchItem) {
	t.Helper()

	if item.ID == "" || strings.Trim(item.ID, "0123456789") != "" {
		t.Errorf("ID = %q, want digits", item.ID)
	}

	if item.Title == "" || !strings.Contains(item.Class, "panel") {
		t.Errorf("item %s: title %q, class %q", item.ID, item.Title, item.Class)
	}

	if len(item.Fields) < 3 || len(item.Fields) > 4 {
		t.Errorf("item %s: fields = %q, want 3 or 4", item.ID, item.Fields)
	}

	if slices.ContainsFunc(item.Fields, func(f string) bool { return strings.Contains(f, "●") }) {
		t.Errorf("item %s: fields %q hold ●", item.ID, item.Fields)
	}
}

// MATSNE-SC-002: A highlight inside a word keeps the word whole.
func TestSearchKeepsAHighlightedWordWhole(t *testing.T) {
	t.Parallel()

	page, err := serve(t, read(t, "search-tax-code.html"), http.StatusOK).
		Search(t.Context(), "ka", matsne.SearchQuery{Query: "საგადასახადო კოდექსი"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	index := slices.IndexFunc(
		page.Items,
		func(item matsne.SearchItem) bool { return item.ID == "4740176" },
	)
	if index < 0 {
		t.Fatal("no item 4740176")
	}

	if title := page.Items[index].Title; !strings.Contains(
		title,
		"საგადასახადო კოდექსის 47-ე მუხლის",
	) {
		t.Errorf("Title = %q, want the word კოდექსის whole", title)
	}
}

// MATSNE-SC-002: An English item has its four fields and no link of a language.
func TestSearchLeavesOutTheLanguageLinks(t *testing.T) {
	t.Parallel()

	page, err := serve(t, read(t, "search-en.html"), http.StatusOK).
		Search(t.Context(), "en", matsne.SearchQuery{Query: "Labour Code"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("Items = %+v, want one", page.Items)
	}

	got := page.Items[0]
	wantFields := []string{
		"Organic Law of Georgia",
		"Parliament of Georgia",
		"N4113-რს",
		"17/12/2010",
	}

	if got.ID != "1155567" ||
		got.Title != "ORGANIC LAW OF GEORGIA – LABOUR CODE OF GEORGIA" ||
		got.Class != "panel panel-success document-search-result-item" ||
		!slices.Equal(got.Fields, wantFields) {
		t.Errorf("item = %+v, want 1155567 with the fields %q", got, wantFields)
	}
}

// MATSNE-SC-002: Suggest decodes the titles.
func TestSuggestReadsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	titles, err := serve(t, read(t, "suggest-tax-code.json"), http.StatusOK).
		Suggest(t.Context(), "საგადასახადო კოდექსი")
	if err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}

	want := matsne.Suggestion{
		ID:    "1043717",
		Title: "საქართველოს საგადასახადო კოდექსი",
		State: "active",
	}
	if len(titles) != 45 || !slices.Contains(titles, want) {
		t.Errorf("Suggest() = %d titles, want 45 with %+v", len(titles), want)
	}
}

// MATSNE-SC-003: The page of the filter, a status that is not a success, and a page with
// no metadata table each give their error.
func TestClientErrors(t *testing.T) {
	t.Parallel()

	blocked := read(t, "blocked.html")
	noTable := []byte(`<html><head><title>x</title></head><body><p>x</p></body></html>`)

	search := func(c *matsne.Client) error {
		_, err := c.Search(t.Context(), "ka", matsne.SearchQuery{Query: "x"})

		return err
	}
	suggest := func(c *matsne.Client) error {
		_, err := c.Suggest(t.Context(), "კოდექსი")

		return err
	}
	document := func(c *matsne.Client) error {
		_, err := c.Document(t.Context(), "ka", "999999999", -1)

		return err
	}

	tests := []struct {
		name   string
		body   []byte
		status int
		call   func(*matsne.Client) error
		want   error
	}{
		{"search blocked", blocked, http.StatusOK, search, matsne.ErrBlocked},
		{"suggest blocked", blocked, http.StatusOK, suggest, matsne.ErrBlocked},
		{"document blocked", blocked, http.StatusOK, document, matsne.ErrBlocked},
		{
			"status 500",
			[]byte("oops"),
			http.StatusInternalServerError,
			document,
			matsne.ErrUnexpectedStatus,
		},
		{"no metadata table", noTable, http.StatusOK, document, matsne.ErrMalformedPage},
		{"no total", noTable, http.StatusOK, search, matsne.ErrMalformedPage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call(serve(t, tt.body, tt.status))
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}

			var apiErr *matsne.APIError
			if tt.status != http.StatusOK &&
				(!errors.As(err, &apiErr) || apiErr.StatusCode != tt.status) {
				t.Errorf("error = %v, want an *APIError with %d", err, tt.status)
			}
		})
	}
}

// MATSNE-SC-004: The client reads the metadata and the versions of the recorded page of
// the Tax Code.
func TestDocumentReadsTheMetadata(t *testing.T) {
	t.Parallel()

	doc, err := serve(t, read(t, "act-1043717-ka.html"), http.StatusOK).
		Document(t.Context(), "ka", "1043717", 248)
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if doc.Title != "საქართველოს საგადასახადო კოდექსი" || doc.Selected != "248" {
		t.Errorf("Title = %q, Selected = %q", doc.Title, doc.Selected)
	}

	if len(doc.Meta) < 6 ||
		doc.Meta[0] != (matsne.MetaRow{Label: "დოკუმენტის ნომერი", Value: "3591"}) {
		t.Errorf("Meta = %+v, want 6 or more rows from the number 3591", doc.Meta)
	}

	first := matsne.Publication{Value: "248", Text: "28/10/2026"}
	last := matsne.Publication{Value: "0", Text: "12/10/2010"}

	if n := len(
		doc.Publications,
	); n != 249 || doc.Publications[0] != first ||
		doc.Publications[n-1] != last {
		t.Errorf("Publications = %d, want 249 from %+v to %+v", n, first, last)
	}
}

// MATSNE-SC-004: The paragraphs of the Georgian page hold no CSS text, and an article
// heading carries its superscript.
func TestDocumentReadsTheParagraphs(t *testing.T) {
	t.Parallel()

	doc, err := serve(t, read(t, "act-1043717-ka.html"), http.StatusOK).
		Document(t.Context(), "ka", "1043717", 248)
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if i := slices.IndexFunc(doc.Paragraphs, func(p matsne.Paragraph) bool {
		return strings.Contains(p.Text, "{")
	}); i >= 0 {
		t.Errorf("paragraph %d holds CSS text: %.80q", i, doc.Paragraphs[i].Text)
	}

	if !slices.ContainsFunc(doc.Paragraphs, func(p matsne.Paragraph) bool {
		return p.Class == "muxlixml" && strings.HasPrefix(p.Text, "მუხლი 133^3.")
	}) {
		t.Error("no paragraph of class muxlixml starts with მუხლი 133^3.")
	}

	for _, want := range []string{"ბ^1) იმ პირის მიერ", "კანონის 34^1 მუხლის შესაბამისად"} {
		if !slices.ContainsFunc(doc.Paragraphs, func(p matsne.Paragraph) bool {
			return strings.Contains(p.Text, want)
		}) {
			t.Errorf("no paragraph holds %q", want)
		}
	}
}

// MATSNE-SC-004: An English page marks its article headings bold, and a version whose
// translation is underway has no paragraph.
func TestDocumentReadsTheEnglishPages(t *testing.T) {
	t.Parallel()

	doc, err := serve(t, read(t, "act-1043717-en.html"), http.StatusOK).
		Document(t.Context(), "en", "1043717", -1)
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if doc.Title != "TAX CODE OF GEORGIA" || doc.Selected != "247" {
		t.Errorf(
			"Title = %q, Selected = %q, want TAX CODE OF GEORGIA and 247",
			doc.Title,
			doc.Selected,
		)
	}

	if !slices.ContainsFunc(doc.Paragraphs, func(p matsne.Paragraph) bool {
		return p.Bold && strings.HasPrefix(p.Text, "Article 133^3 – ")
	}) {
		t.Error("no bold paragraph starts with Article 133^3 – ")
	}

	underway, err := serve(t, read(t, "act-1043717-en-248.html"), http.StatusOK).
		Document(t.Context(), "en", "1043717", 248)
	if err != nil {
		t.Fatalf("Document(248) error = %v", err)
	}

	if len(underway.Paragraphs) != 0 || underway.Selected != "248" {
		t.Errorf("version 248: %d paragraphs, Selected %q, want 0 and 248",
			len(underway.Paragraphs), underway.Selected)
	}
}
