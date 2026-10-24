package gpost_test

import (
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/gpost"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	pagePath   = "/help/postal-codes"
	searchPath = "/Help/FindPostalCode"
	tokenName  = "__RequestVerificationToken" //nolint:gosec // The name of a field, not a token.
	testToken  = "test-token"
)

// recorded is one request that the fake vendor received.
type recorded struct {
	Method string
	Path   string
	Header http.Header
	Form   url.Values
}

// vendor is a fake Georgian Post. It serves page on the page path with the cookie token
// test-token, and result on the search path.
type vendor struct {
	page         []byte
	result       []byte
	pageStatus   int
	searchStatus int
	noCookie     bool

	mu       sync.Mutex
	requests []recorded
}

func (v *vendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	v.mu.Lock()
	v.requests = append(v.requests, recorded{
		Method: r.Method, Path: r.URL.Path, Header: r.Header.Clone(), Form: r.PostForm,
	})
	v.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch r.URL.Path {
	case pagePath:
		if !v.noCookie {
			w.Header().Set("Set-Cookie", tokenName+"="+testToken+"; path=/; HttpOnly")
		}

		if v.pageStatus != 0 {
			w.WriteHeader(v.pageStatus)
		}

		_, _ = w.Write(v.page)
	case searchPath:
		if v.searchStatus != 0 {
			w.WriteHeader(v.searchStatus)
		}

		_, _ = w.Write(v.result)
	default:
		http.NotFound(w, r)
	}
}

func (v *vendor) recorded() []recorded {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.requests
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

func newVendor(t *testing.T, result string) *vendor {
	t.Helper()

	return &vendor{
		page:   readFile(t, "testdata/postal-codes.html"),
		result: readFile(t, "testdata/"+result),
	}
}

func clientOf(t *testing.T, fake *vendor) *gpost.Client {
	t.Helper()

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	return gpost.New(vendorhttp.WithBaseURL(server.URL))
}

// POSTCODE-SC-001: One search sends exactly the visit of the page, then the search, and
// each call names the server and its repository.
func TestPostalCodesSendsTwoCalls(t *testing.T) {
	t.Parallel()

	fake := newVendor(t, "find-none-ka.html")

	if _, err := clientOf(t, fake).PostalCodes(t.Context(), "რუსთავი", "ka"); err != nil {
		t.Fatalf("PostalCodes() error = %v", err)
	}

	requests := fake.recorded()

	calls := make([]string, 0, len(requests))
	for _, request := range requests {
		calls = append(calls, request.Method+" "+request.Path)
	}

	if want := []string{"GET " + pagePath, "POST " + searchPath}; !slices.Equal(calls, want) {
		t.Fatalf("calls = %q, want %q", calls, want)
	}

	for _, request := range requests {
		agent := request.Header.Get("User-Agent")
		if !strings.HasPrefix(agent, "georgia-mcp-gpost/") ||
			!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
			t.Errorf("%s User-Agent = %q, want georgia-mcp-gpost/<version> (+<repository>)",
				request.Path, agent)
		}

		if accept := request.Header.Get("Accept"); accept != "text/html" {
			t.Errorf("%s Accept = %q, want text/html", request.Path, accept)
		}
	}
}

// POSTCODE-SC-002: The search carries the form token, the region type, the text, and
// the cookie token with the language.
func TestPostalCodesSendsTheTokens(t *testing.T) {
	t.Parallel()

	fake := newVendor(t, "find-none-ka.html")

	if _, err := clientOf(t, fake).PostalCodes(t.Context(), "რუსთავი", "ka"); err != nil {
		t.Fatalf("PostalCodes() error = %v", err)
	}

	search := fake.recorded()[1]

	wantForm := url.Values{
		tokenName:               {testToken},
		"regionTypeId":          {"1"},
		"postalCodeSearchValue": {"რუსთავი"},
	}
	if !maps.EqualFunc(search.Form, wantForm, slices.Equal) {
		t.Errorf("form = %q, want %q", search.Form, wantForm)
	}

	wantHeader := map[string]string{
		"Cookie":           tokenName + "=" + testToken + "; lang=ka",
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Requested-With": "XMLHttpRequest",
		"Referer":          gpost.DefaultBaseURL + pagePath,
	}
	for key, want := range wantHeader {
		if got := search.Header.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

// POSTCODE-SC-003: The client reads each row of a recorded answer, hidden rows too, with
// the district of each.
func TestPostalCodesReadsARecordedAnswer(t *testing.T) {
	t.Parallel()

	got, err := clientOf(t, newVendor(t, "find-chavchavadze-ka.html")).
		PostalCodes(t.Context(), "ჭავჭავაძის", "ka")
	if err != nil {
		t.Fatalf("PostalCodes() error = %v", err)
	}

	if len(got) != 63 {
		t.Fatalf("len(PostalCodes()) = %d, want 63", len(got))
	}

	want := map[int]gpost.PostalCodeRow{
		0: {
			District: "თბილისი",
			Code:     "0108",
			Address:  "ქალაქი თბილისი, ალექსანდრე ჭავჭავაძის I ჩიხი",
		},
		6:  {District: "თბილისი", Code: "0179", Address: "ქალაქი თბილისი, ილია ჭავჭავაძის II ჩიხი"},
		55: {District: "ბათუმი", Code: "6010", Address: "ქალაქი ბათუმი, ილია ჭავჭავაძის I ჩიხი"},
	}
	for index, row := range want {
		if got[index] != row {
			t.Errorf("row %d = %q, want %q", index+1, got[index], row)
		}
	}

	for index := 55; index < 59; index++ {
		if got[index].District != "ბათუმი" {
			t.Errorf("row %d district = %q, want ბათუმი", index+1, got[index].District)
		}
	}
}

// POSTCODE-SC-004: An answer with no match gives an empty list and no error.
func TestPostalCodesReadsAnEmptyAnswer(t *testing.T) {
	t.Parallel()

	got, err := clientOf(t, newVendor(t, "find-none-ka.html")).
		PostalCodes(t.Context(), "ზზზზზქქ", "ka")
	if err != nil {
		t.Fatalf("PostalCodes() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("PostalCodes() = %q, want no row", got)
	}
}

// POSTCODE-SC-005: The language goes in the cookie of both calls, and the answer in
// English keeps its text.
func TestPostalCodesSendsTheLanguage(t *testing.T) {
	t.Parallel()

	fake := newVendor(t, "find-rustavi-en.html")

	got, err := clientOf(t, fake).PostalCodes(t.Context(), "Rustavi", "en")
	if err != nil {
		t.Fatalf("PostalCodes() error = %v", err)
	}

	requests := fake.recorded()
	if cookie := requests[0].Header.Get("Cookie"); cookie != "lang=en" {
		t.Errorf("page Cookie = %q, want lang=en", cookie)
	}

	if cookie := requests[1].Header.Get("Cookie"); !strings.HasSuffix(cookie, "; lang=en") {
		t.Errorf("search Cookie = %q, want the suffix ; lang=en", cookie)
	}

	if len(got) != 12 {
		t.Fatalf("len(PostalCodes()) = %d, want 12", len(got))
	}

	want := gpost.PostalCodeRow{District: "Rustavi City", Code: "3700", Address: "City Rustavi"}
	if got[0] != want {
		t.Errorf("first row = %q, want %q", got[0], want)
	}
}

// POSTCODE-SC-006: A page with no form token, or an answer with no cookie token, gives
// ErrNoToken and sends no search.
func TestPostalCodesNeedsTheTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setUp func(*vendor)
	}{
		{
			name:  "no form",
			setUp: func(v *vendor) { v.page = readFile(t, "testdata/postal-codes-no-form.html") },
		},
		{name: "no cookie", setUp: func(v *vendor) { v.noCookie = true }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fake := newVendor(t, "find-none-ka.html")
			test.setUp(fake)

			_, err := clientOf(t, fake).PostalCodes(t.Context(), "რუსთავი", "ka")
			if !errors.Is(err, gpost.ErrNoToken) {
				t.Errorf("PostalCodes() error = %v, want ErrNoToken", err)
			}

			if count := len(fake.recorded()); count != 1 {
				t.Errorf("calls = %d, want 1", count)
			}
		})
	}
}

// POSTCODE-SC-007: An answer that the client cannot read gives ErrMalformedResult.
func TestPostalCodesRefusesAMalformedAnswer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result []byte
	}{
		{name: "a row with no code", result: readFile(t, "testdata/find-malformed.html")},
		{
			name: "a row before a district",
			result: []byte(
				`<div class="com-cell-postcode"><div class="com-postcode-name">0186</div>` +
					`<div class="com-location-name">ქალაქი თბილისი</div></div>`,
			),
		},
		{
			name: "a row with no address",
			result: []byte(`<div class="com-district">თბილისი</div>` +
				`<div class="com-cell-postcode"><div class="com-postcode-name">0186</div></div>`),
		},
		{name: "no row and no note", result: []byte(`<div class="changed">0186</div>`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fake := newVendor(t, "find-none-ka.html")
			fake.result = test.result

			_, err := clientOf(t, fake).PostalCodes(t.Context(), "რუსთავი", "ka")
			if !errors.Is(err, gpost.ErrMalformedResult) {
				t.Errorf("PostalCodes() error = %v, want ErrMalformedResult", err)
			}
		})
	}
}

// POSTCODE-SC-008: A status other than 200 on either call is an APIError.
func TestPostalCodesReportsAStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setUp func(*vendor)
		calls int
	}{
		{
			name:  "page",
			setUp: func(v *vendor) { v.pageStatus = http.StatusInternalServerError },
			calls: 1,
		},
		{
			name:  "search",
			setUp: func(v *vendor) { v.searchStatus = http.StatusInternalServerError },
			calls: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fake := newVendor(t, "find-none-ka.html")
			test.setUp(fake)

			_, err := clientOf(t, fake).PostalCodes(t.Context(), "რუსთავი", "ka")

			var apiErr *gpost.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
				t.Fatalf("PostalCodes() error = %v, want an APIError with 500", err)
			}

			if !errors.Is(err, gpost.ErrUnexpectedStatus) {
				t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false, want true")
			}

			if count := len(fake.recorded()); count != test.calls {
				t.Errorf("calls = %d, want %d", count, test.calls)
			}
		})
	}
}
