package liberty_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/liberty"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const english = "en"

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// recorder keeps the last request and its body.
type recorder struct {
	request *http.Request
	body    []byte
}

func recordingServer(t *testing.T, answer []byte) (*httptest.Server, *recorder) {
	t.Helper()

	got := &recorder{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.request = r.Clone(r.Context())
		got.body, _ = io.ReadAll(r.Body)

		_, _ = w.Write(answer)
	}))
	t.Cleanup(server.Close)

	return server, got
}

func checkAgent(t *testing.T, request *http.Request, accept string) {
	t.Helper()

	if got := request.Header.Get("Accept"); got != accept {
		t.Errorf("Accept = %q, want %q", got, accept)
	}

	agent := request.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}
}

// LIBOFFER-SC-001: A page of offers sends the path and the body, with null for no filter.
func TestOffersSendsTheRequest(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, mustRead(t, "testdata/offers-1.json"))

	offers, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), liberty.OffersQuery{Page: 1, Language: english})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if got.request.Method != http.MethodPost ||
		got.request.URL.Path != "/_mvcapi/offersapi/loadOffers" {
		t.Errorf("request = %s %s", got.request.Method, got.request.URL.Path)
	}

	want := `{"categoryId":null,"subCategoryId":null,"typeId":null,"partnerId":null,` +
		`"locationId":null,"sortBy":"","page":1,"langAbbr":"en"}`
	if string(got.body) != want {
		t.Errorf("body = %s, want %s", got.body, want)
	}

	checkAgent(t, got.request, "application/json")

	if len(offers) != 16 {
		t.Errorf("Offers() = %d offers, want 16", len(offers))
	}
}

// LIBOFFER-SC-001: A page of a category or a type sends its id.
func TestOffersSendsAFilter(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, mustRead(t, "testdata/category-6-1.json"))

	_, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), liberty.OffersQuery{CategoryID: 6, TypeID: 2, Page: 3, Language: "ka"})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	var body struct {
		CategoryID *int   `json:"categoryId"`
		TypeID     *int   `json:"typeId"`
		Page       int    `json:"page"`
		LangAbbr   string `json:"langAbbr"`
	}
	if err := json.Unmarshal(got.body, &body); err != nil {
		t.Fatal(err)
	}

	if body.CategoryID == nil || *body.CategoryID != 6 || body.TypeID == nil || *body.TypeID != 2 ||
		body.Page != 3 || body.LangAbbr != "ka" {
		t.Errorf("body = %s", got.body)
	}
}

// LIBOFFER-SC-001: The offer 727 decodes with its labels and its days left.
func TestOffersDecodesAnOffer(t *testing.T) {
	t.Parallel()

	server, _ := recordingServer(t, mustRead(t, "testdata/offers-1.json"))

	offers, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), liberty.OffersQuery{Page: 1, Language: english})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	index := slices.IndexFunc(offers, func(offer liberty.Offer) bool { return offer.ID == 727 })
	if index < 0 {
		t.Fatal("the page holds no offer 727")
	}

	hyatt := offers[index]
	if !strings.HasPrefix(hyatt.Title, "Hyatt Regency Bangkok Sukhumvit") ||
		hyatt.RemainingDays != "72 Days left" ||
		!slices.Equal(hyatt.Labels, []string{"Special Hotel Offer"}) {
		t.Errorf("offer 727 = %+v", hyatt)
	}
}

// LIBOFFER-SC-005: The categories decode with their ids and titles.
func TestCategories(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, mustRead(t, "testdata/categories.json"))

	categories, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).
		Categories(t.Context(), english)
	if err != nil {
		t.Fatalf("Categories() error = %v", err)
	}

	if got.request.URL.Path != "/_mvcapi/offersapi/loadcategories" ||
		string(got.body) != `{"langAbbr":"en"}` {
		t.Errorf("request = %s %s", got.request.URL.Path, got.body)
	}

	if len(categories) != 12 || categories[0] != (liberty.Category{ID: 1, Title: "Auto-moto"}) {
		t.Errorf("categories = %+v", categories)
	}
}

// LIBOFFER-SC-005: The types decode with their ids and titles.
func TestTypes(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, mustRead(t, "testdata/types.json"))

	types, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).Types(t.Context(), english)
	if err != nil {
		t.Fatalf("Types() error = %v", err)
	}

	if got.request.URL.Path != "/_mvcapi/offersapi/loadtypes" {
		t.Errorf("path = %s", got.request.URL.Path)
	}

	if len(types) != 5 || types[1] != (liberty.Type{ID: 2, Title: "Cashback"}) {
		t.Errorf("types = %+v", types)
	}
}

// LIBOFFER-SC-002: An answer that reports no success is an error.
func TestOffersReportsNoSuccess(t *testing.T) {
	t.Parallel()

	server, _ := recordingServer(
		t,
		[]byte(`{"data":{"list":[]},"succeeded":false,"message":"error"}`),
	)

	_, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), liberty.OffersQuery{Page: 1})
	if !errors.Is(err, liberty.ErrNoSuccess) {
		t.Errorf("Offers() error = %v, want ErrNoSuccess", err)
	}
}

// LIBOFFER-SC-009: A status other than 200 is an APIError with the first 1 KiB of the body.
func TestReportsAStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
	}))
	t.Cleanup(server.Close)

	_, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).Types(t.Context(), english)

	apiErr, ok := errors.AsType[*liberty.APIError](err)
	if !ok {
		t.Fatalf("Types() error = %v, want an *APIError", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError || len(apiErr.Body) != 1024 {
		t.Errorf("APIError = %d with %d bytes", apiErr.StatusCode, len(apiErr.Body))
	}

	if !errors.Is(err, liberty.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(%v, ErrUnexpectedStatus) = false", err)
	}
}

// LIBOFFER-SC-003: The page of offer 727 gives its title, its days left, and its 13
// paragraphs. A page with no block of terms is ErrNoTerms.
func TestOfferPage(t *testing.T) {
	t.Parallel()

	server, got := recordingServer(t, mustRead(t, "testdata/details-727.html"))

	page, err := liberty.New(vendorhttp.WithBaseURL(server.URL)).
		OfferPage(t.Context(), 727, english)
	if err != nil {
		t.Fatalf("OfferPage() error = %v", err)
	}

	if got.request.Method != http.MethodGet ||
		got.request.URL.Path != "/en/studio-lifestyle/offers/details/727" {
		t.Errorf("request = %s %s", got.request.Method, got.request.URL.Path)
	}

	checkAgent(t, got.request, "text/html")

	if page.Title != "Hyatt Regency Bangkok Sukhumvit" || page.Timer != "72 Days left" ||
		len(page.Paragraphs) != 13 ||
		page.Paragraphs[12] != "• The offer is valid until December 31, 2026" {
		t.Errorf("page = %q, %q, %d paragraphs", page.Title, page.Timer, len(page.Paragraphs))
	}

	emptyServer, _ := recordingServer(
		t,
		[]byte(`<html><body><h1 class="featured-hero__title">X</h1></body></html>`),
	)

	_, err = liberty.New(vendorhttp.WithBaseURL(emptyServer.URL)).
		OfferPage(t.Context(), 727, english)
	if !errors.Is(err, liberty.ErrNoTerms) {
		t.Errorf("OfferPage(no terms) error = %v, want ErrNoTerms", err)
	}
}
