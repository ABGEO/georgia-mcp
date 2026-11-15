package tas_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/pkg/tas"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	freedomCode   = "01.18.04.020.005"
	approvedNo    = "AR11168188"
	searchCode    = "search-code.js"
	searchNumber  = "search-number.js"
	plainTextType = "text/plain"
)

func tbilisi() *time.Location {
	return time.FixedZone("Tbilisi", 4*60*60)
}

type recorded struct {
	method, path, contentType, agent, body string
}

type requestLog struct {
	mu       sync.Mutex
	requests []recorded
}

func (l *requestLog) last(t *testing.T) recorded {
	t.Helper()

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.requests) == 0 {
		t.Fatal("no request")
	}

	return l.requests[len(l.requests)-1]
}

func (l *requestLog) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.requests)
}

// serve answers each request with the body and the status, and records the request.
func serve(t *testing.T, body string, status int) (*tas.Client, *requestLog) {
	t.Helper()

	requests := &requestLog{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent, _ := io.ReadAll(r.Body)

		requests.mu.Lock()
		requests.requests = append(requests.requests, recorded{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			agent:       r.Header.Get("User-Agent"),
			body:        string(sent),
		})
		requests.mu.Unlock()

		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	return tas.New(vendorhttp.WithBaseURL(server.URL)), requests
}

func serveFile(t *testing.T, file string) (*tas.Client, *requestLog) {
	t.Helper()

	body, err := os.ReadFile("testdata/" + file) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return serve(t, string(body), http.StatusOK)
}

// TASPERM-SC-001: The client sends the User-Agent of the server, a text body, and the body
// of the specification for a code and two statuses.
func TestSearchDocumentsSendsTheRequest(t *testing.T) {
	t.Parallel()

	client, requests := serveFile(t, searchCode)

	_, err := client.SearchDocuments(t.Context(), tas.Filter{
		CadastralCode: freedomCode,
		StatusIDs:     []int{1, 2},
		Limit:         500,
	})
	if err != nil {
		t.Fatalf("SearchDocuments() error = %v", err)
	}

	got := requests.last(t)

	path := "/architect/dwr/call/plaincall/DocumentManager.getDocsForPublicInfo.dwr"
	if got.method != http.MethodPost || got.path != path || got.contentType != plainTextType {
		t.Errorf("request = %s %s %s, want POST %s %s", got.method, got.path, got.contentType,
			path, plainTextType)
	}

	if !strings.HasPrefix(got.agent, "georgia-mcp/") ||
		!strings.Contains(got.agent, "https://github.com/abgeo/georgia-mcp") {
		t.Errorf("User-Agent = %q, want the agent of Georgia MCP", got.agent)
	}

	want := []string{
		"callCount=1",
		"windowName=",
		"nextReverseAjaxIndex=0",
		"c0-scriptName=DocumentManager",
		"c0-methodName=getDocsForPublicInfo",
		"c0-id=0",
		"c0-e1=number:2",
		"c0-e2=number:0",
		"c0-e3=number:500",
		"c0-e4=string:01.18.04.020.005",
		"c0-e5=Array:[reference:c0-e6,reference:c0-e7]",
		"c0-e6=number:1",
		"c0-e7=number:2",
		"c0-param0=Object_Object:{applicationId:reference:c0-e1, start:reference:c0-e2, " +
			"limit:reference:c0-e3, naprCadCode:reference:c0-e4, docStatusIds:reference:c0-e5}",
		"batchId=1",
		"instanceId=0",
		"page=%2Farchitect%2FpublicInformation.html",
		"httpSessionId=",
		"scriptSessionId=00000000000000000000000000000000/georgia-mcp",
	}

	lines := strings.Split(strings.TrimSuffix(got.body, "\n"), "\n")
	if !slices.Equal(lines, want) {
		t.Errorf("body =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

// TASPERM-SC-002: A start of the period goes as a date in milliseconds.
func TestSearchDocumentsSendsADate(t *testing.T) {
	t.Parallel()

	client, requests := serveFile(t, searchCode)

	_, err := client.SearchDocuments(t.Context(), tas.Filter{
		From:  time.Date(2026, time.November, 4, 0, 0, 0, 0, tbilisi()),
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("SearchDocuments() error = %v", err)
	}

	body := requests.last(t).body
	if !strings.Contains(body, "\nc0-e4=Date:1793736000000\n") ||
		!strings.Contains(body, "fromDate:reference:c0-e4}") {
		t.Errorf("body = %q, want fromDate as c0-e4=Date:1793736000000", body)
	}
}

// TASPERM-SC-002: A text that would change the body, or a limit below 1, sends no call.
func TestSearchDocumentsRefusesAFilter(t *testing.T) {
	t.Parallel()

	filters := map[string]tas.Filter{
		"code":   {CadastralCode: "01.18;x", Limit: 5},
		"number": {DocumentNo: "AR1\nx", Limit: 5},
		"limit":  {CadastralCode: freedomCode},
		"start":  {CadastralCode: freedomCode, Start: -1, Limit: 5},
	}

	for name, filter := range filters {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, requests := serveFile(t, searchCode)

			if _, err := client.SearchDocuments(
				t.Context(),
				filter,
			); !errors.Is(
				err,
				tas.ErrInvalidFilter,
			) {
				t.Errorf("SearchDocuments() error = %v, want ErrInvalidFilter", err)
			}

			if requests.count() != 0 {
				t.Errorf("calls = %d, want 0", requests.count())
			}
		})
	}
}

// TASPERM-SC-003: The answer of a code gives its total and its applications.
func TestSearchDocumentsReadsACode(t *testing.T) {
	t.Parallel()

	client, _ := serveFile(t, searchCode)

	page, err := client.SearchDocuments(
		t.Context(),
		tas.Filter{CadastralCode: freedomCode, Limit: 500},
	)
	if err != nil {
		t.Fatalf("SearchDocuments() error = %v", err)
	}

	if page.Total != 5 || len(page.Documents) != 5 {
		t.Fatalf("page = %d of %d, want 5 of 5", len(page.Documents), page.Total)
	}

	first := page.Documents[0]
	if !strings.HasPrefix(first.DocumentNo, "AR1") || first.RegistrationDate == nil ||
		first.CachedInfo.Nomenclature.CategoryName == "" {
		t.Errorf("Documents[0] = %+v, want a number, a date, and a category", first)
	}
}

// TASPERM-SC-003: An unknown code gives no application.
func TestSearchDocumentsReadsAnEmptyAnswer(t *testing.T) {
	t.Parallel()

	client, _ := serveFile(t, "search-empty.js")

	page, err := client.SearchDocuments(
		t.Context(),
		tas.Filter{CadastralCode: "99.99.99.999.999", Limit: 5},
	)
	if err != nil || page.Total != 0 || len(page.Documents) != 0 {
		t.Errorf("SearchDocuments() = %+v, %v, want an empty page", page, err)
	}
}

// TASPERM-SC-004: The answer of a number gives the application with its executor and its
// decision.
func TestSearchDocumentsReadsANumber(t *testing.T) {
	t.Parallel()

	client, _ := serveFile(t, searchNumber)

	page, err := client.SearchDocuments(t.Context(), tas.Filter{DocumentNo: approvedNo, Limit: 50})
	if err != nil || len(page.Documents) != 1 {
		t.Fatalf("SearchDocuments() = %+v, %v, want 1 application", page, err)
	}

	got := page.Documents[0]
	if got.DocumentNo != approvedNo || got.DocumentID != 1168188 || got.DocumentStatusID != 1 ||
		got.CachedInfo.ExecutorName != "ეკა მხეიძე" || !strings.HasPrefix(got.ResponseText, "<p>") {
		t.Errorf("Documents[0] = %+v", got)
	}

	registered := got.RegistrationDate.Time().In(tbilisi()).Format(time.DateOnly)
	if registered != "2026-11-04" {
		t.Errorf("RegistrationDate = %s, want 2026-11-04", registered)
	}
}

// TASPERM-SC-005: An exception, a reply with no callback, a failure, and a status that is
// not a success are errors.
func TestSearchDocumentsFails(t *testing.T) {
	t.Parallel()

	exception, err := os.ReadFile("testdata/exception.js")
	if err != nil {
		t.Fatal(err)
	}

	failed := `//#DWR-REPLY` + "\n" +
		`dwr.engine.remote.handleCallback("1","0",{errorMessage:"no",isSuccess:false,source:null,sources:null});`

	tests := []struct {
		name, body string
		status     int
		want       error
		holds      string
	}{
		{"exception", string(exception), http.StatusOK, tas.ErrRemote, "IllegalArgumentException"},
		{"no callback", "<html></html>", http.StatusOK, tas.ErrInvalidReply, ""},
		{"not a success", failed, http.StatusOK, tas.ErrRemote, "no"},
		{"status", "", http.StatusInternalServerError, tas.ErrUnexpectedStatus, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, _ := serve(t, tt.body, tt.status)

			_, err := client.SearchDocuments(
				t.Context(),
				tas.Filter{CadastralCode: freedomCode, Limit: 5},
			)
			if !errors.Is(err, tt.want) || !strings.Contains(err.Error(), tt.holds) {
				t.Errorf("SearchDocuments() error = %v, want %v holding %q", err, tt.want, tt.holds)
			}
		})
	}
}
