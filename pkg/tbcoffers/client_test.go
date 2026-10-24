package tbcoffers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	oktopusID = "7tD3DL0YuO2V6f1Efr4sSV"
	locale    = "en-US"
)

func serveFile(t *testing.T, path string) *httptest.Server {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server
}

// recorder keeps the last request and its body.
type recorder struct {
	request *http.Request
	body    []byte
}

func recordingServer(t *testing.T, answer string) (*httptest.Server, *recorder) {
	t.Helper()

	got := &recorder{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.request = r.Clone(r.Context())
		got.body, _ = io.ReadAll(r.Body)

		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)

	return server, got
}

func checkHeaders(t *testing.T, request *http.Request) {
	t.Helper()

	if accept := request.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := request.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}
}

// TBCOFFER-SC-001: The list sends the body with no segment and an empty filter list.
func TestOffersSendsTheRequest(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, `{"pagingDetails":{},"list":[]}`)

	_, err := tbcoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), tbcoffers.OffersRequest{Locale: locale, PageSize: 1000})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if got.request.Method != http.MethodPost || got.request.URL.Path != "/entries/offer" {
		t.Errorf(
			"request = %s %s, want POST /entries/offer",
			got.request.Method,
			got.request.URL.Path,
		)
	}

	if body := string(
		got.body,
	); body != `{"filter":[],"locale":"en-US","pageIndex":0,"pageSize":1000}` {
		t.Errorf("body = %s", body)
	}

	if contentType := got.request.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	checkHeaders(t, got.request)
}

// TBCOFFER-SC-001: The list decodes the 518 entries of a recorded answer.
func TestOffersDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/offers.json")

	page, err := tbcoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), tbcoffers.OffersRequest{Locale: locale, PageSize: 1000})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if page.PagingDetails.TotalCount != 518 || len(page.List) != 518 ||
		page.PagingDetails.HasNextPage {
		t.Fatalf("page = %d of %d, next %v; want 518 of 518, no next",
			len(page.List), page.PagingDetails.TotalCount, page.PagingDetails.HasNextPage)
	}

	index := slices.IndexFunc(
		page.List,
		func(entry tbcoffers.Entry) bool { return entry.ID == oktopusID },
	)
	if index < 0 {
		t.Fatal("the answer holds no Oktopus")
	}

	type digest struct {
		Title, Partner, Slug, Start string
		Segments                    int
	}

	entry := page.List[index]
	got := digest{entry.Title, "", entry.Slug, entry.StartDate, len(entry.Segments)}

	if entry.Partner != nil {
		got.Partner = entry.Partner.Title
	}

	want := digest{"Oktopus", "Oktopus Fitness Club", "oktopusi-qeshbeqi", "2026-10-20T00:00:00", 2}
	if got != want {
		t.Errorf("Oktopus = %+v, want %+v", got, want)
	}
}

// TBCOFFER-SC-001: A filtered list sends its filter.
func TestOffersSendsAFilter(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, `{"pagingDetails":{},"list":[]}`)

	_, err := tbcoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), tbcoffers.OffersRequest{
			Filter: []string{"Category:Leisure"}, Locale: locale, PageSize: 1000,
		})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	var body struct {
		Filter []string `json:"filter"`
	}
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(body.Filter, []string{"Category:Leisure"}) {
		t.Errorf("filter = %v, want [Category:Leisure]", body.Filter)
	}
}

// TBCOFFER-SC-001: The filters of a segment send the type, the segment, and the locale.
func TestFiltersOfASegment(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, string(mustRead(t, "testdata/filters-concept.json")))

	filters, err := tbcoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Filters(t.Context(), "Concept", locale)
	if err != nil {
		t.Fatalf("Filters() error = %v", err)
	}

	want := url.Values{"type": {"offer"}, "segment": {"Concept"}, "locale": {locale}}
	if got.request.URL.Path != "/filters" || got.request.URL.Query().Encode() != want.Encode() {
		t.Errorf("request = %s", got.request.URL)
	}

	if len(filters) != 4 || filters[0].Code != "Category" || len(filters[0].Childs) == 0 {
		t.Errorf("filters = %+v, want 4 groups from Category", filters)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// TBCOFFER-SC-002: The page of an offer sends its slug and locale, and decodes its tree.
func TestOfferReadsThePage(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, string(mustRead(t, "testdata/offer-oktopusi-qeshbeqi.json")))

	entry, err := tbcoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offer(t.Context(), "oktopusi-qeshbeqi", locale)
	if err != nil {
		t.Fatalf("Offer() error = %v", err)
	}

	if got.request.Method != http.MethodGet ||
		got.request.URL.Path != "/entries/offer/oktopusi-qeshbeqi/en-US" {
		t.Errorf("request = %s %s", got.request.Method, got.request.URL.Path)
	}

	checkHeaders(t, got.request)

	if entry.ID != oktopusID || entry.Content == nil || entry.Content.NodeType != "document" ||
		len(entry.Content.Content) == 0 {
		t.Errorf("entry = %+v, want Oktopus with a document", entry)
	}
}

// TBCOFFER-SC-008: A status other than 200 is an APIError with the first 1 KiB of the body.
func TestReportsAStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
	}))
	t.Cleanup(server.Close)

	_, err := tbcoffers.New(vendorhttp.WithBaseURL(server.URL)).Filters(t.Context(), "", locale)

	apiErr, ok := errors.AsType[*tbcoffers.APIError](err)
	if !ok {
		t.Fatalf("Filters() error = %v, want an *APIError", err)
	}

	if apiErr.StatusCode != http.StatusBadRequest || len(apiErr.Body) != 1024 {
		t.Errorf("APIError = %d with %d bytes", apiErr.StatusCode, len(apiErr.Body))
	}

	if !errors.Is(err, tbcoffers.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(%v, ErrUnexpectedStatus) = false", err)
	}
}
