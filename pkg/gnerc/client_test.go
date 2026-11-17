package gnerc_test

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

	"github.com/abgeo/georgia-mcp/pkg/gnerc"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	balanceName  = "2015 - 2026 ფაქტიური ბალანსი.xlsx"
	lastModified = "Wed, 21 Oct 2026 09:45:18 GMT"
	homeFile     = "home.html"
	balanceFile  = "balance.xlsx"
)

type answer struct {
	status int
	file   string
}

type recorded struct {
	path      string
	userAgent string
}

// serve answers each escaped path of files with its answer, and any other path with 404.
// It records each request.
func serve(t *testing.T, files map[string]answer) (*gnerc.Client, func() []recorded) {
	t.Helper()

	var (
		mu    sync.Mutex
		asked []recorded
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := recorded{path: r.URL.EscapedPath(), userAgent: r.UserAgent()}

		mu.Lock()
		defer mu.Unlock()

		asked = append(asked, request)

		reply, ok := files[r.URL.EscapedPath()]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		body, err := os.ReadFile("testdata/" + reply.file)
		if err != nil {
			t.Error(err)

			return
		}

		w.Header().Set("Last-Modified", lastModified)
		w.WriteHeader(reply.status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	client := gnerc.New(vendorhttp.WithBaseURL(server.URL))

	return client, func() []recorded {
		mu.Lock()
		defer mu.Unlock()

		return append([]recorded(nil), asked...)
	}
}

func excel(name string) string {
	return "/excel/" + url.PathEscape(name)
}

// GNERCMON-SC-001: Links sends GET /. Workbook sends the escaped path of the name. Each
// call names Georgia MCP and its repository.
func TestClientSendsTheRequests(t *testing.T) {
	t.Parallel()

	client, asked := serve(t, map[string]answer{
		"/":                {http.StatusOK, homeFile},
		excel(balanceName): {http.StatusOK, balanceFile},
	})

	if _, err := client.Links(t.Context()); err != nil {
		t.Fatalf("Links() error = %v", err)
	}

	if _, err := client.Workbook(t.Context(), balanceName); err != nil {
		t.Fatalf("Workbook() error = %v", err)
	}

	want := "/excel/2015%20-%202026%20%E1%83%A4%E1%83%90%E1%83%A5%E1%83%A2%E1%83%98%E1%83%A3" +
		"%E1%83%A0%E1%83%98%20%E1%83%91%E1%83%90%E1%83%9A%E1%83%90%E1%83%9C%E1%83%A1%E1%83%98.xlsx"

	requests := asked()
	if len(requests) != 2 || requests[0].path != "/" || requests[1].path != want {
		t.Fatalf("requests = %+v, want / and %s", requests, want)
	}

	for _, request := range requests {
		if !strings.HasPrefix(request.userAgent, "georgia-mcp/") ||
			!strings.Contains(request.userAgent, "https://github.com/abgeo/georgia-mcp") {
			t.Errorf("User-Agent = %q, want georgia-mcp and its repository", request.userAgent)
		}
	}
}

// GNERCMON-SC-002: Links gives the 7 links of the folder excel, the balance first, with
// their text.
func TestClientReadsTheLinks(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, map[string]answer{"/": {http.StatusOK, homeFile}})

	links, err := client.Links(t.Context())
	if err != nil {
		t.Fatalf("Links() error = %v", err)
	}

	var workbooks []gnerc.Link

	for _, link := range links {
		if strings.Contains(link.Href, "/excel/") {
			workbooks = append(workbooks, link)
		}
	}

	if len(workbooks) != 7 {
		t.Fatalf("links of excel = %+v, want 7", workbooks)
	}

	want := gnerc.Link{
		Href: "https://data.gnerc.org/excel/" + balanceName,
		Text: "დეტალური მონაცემების გადმოწერა",
	}
	if workbooks[0] != want {
		t.Errorf("first link = %+v, want %+v", workbooks[0], want)
	}
}

func row(t *testing.T, sheet gnerc.Sheet, number int) gnerc.Row {
	t.Helper()

	for _, r := range sheet.Rows {
		if r.Number == number {
			return r
		}
	}

	t.Fatalf("sheet %s has no row %d", sheet.Name, number)

	return gnerc.Row{}
}

// GNERCMON-SC-003: Workbook gives the sheets in order, the header Last-Modified, and each
// cell as its text.
func TestClientReadsTheBalance(t *testing.T) {
	t.Parallel()

	client, _ := serve(t, map[string]answer{excel(balanceName): {http.StatusOK, balanceFile}})

	book, err := client.Workbook(t.Context(), balanceName)
	if err != nil {
		t.Fatalf("Workbook() error = %v", err)
	}

	if book.LastModified != lastModified {
		t.Errorf("LastModified = %q, want %q", book.LastModified, lastModified)
	}

	names := make([]string, 0, len(book.Sheets))
	for _, sheet := range book.Sheets {
		names = append(names, sheet.Name)
	}

	wantNames := []string{
		"2015 GEO", "2016 GEO", "2017 GEO", "2018 GEO", "2019 GEO", "2020 GEO",
		"2021 GEO", "2022 GEO", "2023 GEO", "2024 GEO", "2025 GEO", "2026 GEO",
	}
	if !slices.Equal(wantNames, names) {
		t.Fatalf("sheets = %v, want %v", names, wantNames)
	}

	wantCells := []gnerc.Cell{
		{Column: "A", Text: "1"},
		{Column: "B", Type: "s", Text: "ელექტროენერგიის ჯამური გამომუშავება"},
		{Column: "C", Text: "910.16262400000005"},
	}
	if got := row(t, book.Sheets[11], 4).Cells[:3]; !slices.Equal(wantCells, got) {
		t.Errorf("row 4 of 2026 = %+v, want %+v", got, wantCells)
	}
}

// GNERCMON-SC-003: A shared string keeps its non-breaking spaces.
func TestClientReadsTheText(t *testing.T) {
	t.Parallel()

	const name = "დერეგულირებული სადგურების ფასები 2015 - 2026 წლები.xlsx"

	client, _ := serve(t, map[string]answer{excel(name): {http.StatusOK, "deregulated.xlsx"}})

	book, err := client.Workbook(t.Context(), name)
	if err != nil {
		t.Fatalf("Workbook() error = %v", err)
	}

	for _, cell := range row(t, book.Sheets[0], 7).Cells {
		if cell.Column == "H" {
			if cell.Text != " 12.304 " || cell.Type != "s" {
				t.Errorf("H7 = %+v, want the shared string \\u00a012.304\\u00a0", cell)
			}

			return
		}
	}

	t.Error("row 7 has no cell H")
}

// GNERCMON-SC-004: A status that is not 200 is an *APIError. A body that is not a zip is
// ErrNotWorkbook.
func TestClientReportsFailures(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		client, _ := serve(t, map[string]answer{
			"/": {status, homeFile}, excel(balanceName): {status, balanceFile},
		})

		_, err := client.Links(t.Context())
		checkStatus(t, err, status)

		_, err = client.Workbook(t.Context(), balanceName)
		checkStatus(t, err, status)
	}

	client, _ := serve(t, map[string]answer{excel(balanceName): {http.StatusOK, "not-zip.xlsx"}})

	if _, err := client.Workbook(t.Context(), balanceName); !errors.Is(err, gnerc.ErrNotWorkbook) {
		t.Errorf("Workbook() of a page error = %v, want ErrNotWorkbook", err)
	}
}

func checkStatus(t *testing.T, err error, status int) {
	t.Helper()

	var apiErr *gnerc.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != status ||
		!errors.Is(err, gnerc.ErrUnexpectedStatus) {
		t.Errorf("error = %v, want an *APIError with %d", err, status)
	}
}
