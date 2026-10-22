package bogoffers_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/bogoffers"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func serveFile(t *testing.T, path string, status int) *httptest.Server {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server
}

// BOGOFFER-SC-001: The client sends the method, the path, the query, the body, and the headers.
func TestOffersSendsTheRequest(t *testing.T) {
	t.Parallel()

	var (
		got     *http.Request
		gotBody string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		_, _ = w.Write([]byte(`{"code":0,"result":{"offers":[]}}`))
	}))
	t.Cleanup(server.Close)

	client := bogoffers.New(vendorhttp.WithBaseURL(server.URL))

	_, err := client.Offers(
		t.Context(),
		bogoffers.OffersQuery{Language: "en", PageNumber: 2, PageSize: 100},
	)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	checkRequest(t, got, gotBody)
}

func checkRequest(t *testing.T, got *http.Request, gotBody string) {
	t.Helper()

	if got.Method != http.MethodPost || got.URL.Path != "/get-offers" {
		t.Errorf("request = %s %s, want POST /get-offers", got.Method, got.URL.Path)
	}

	want := url.Values{"pageInfo": {"true"}, "pageNumber": {"2"}, "pageSize": {"100"}}
	if got.URL.Query().Encode() != want.Encode() {
		t.Errorf("query = %q, want %q", got.URL.Query().Encode(), want.Encode())
	}

	if gotBody != "{}" {
		t.Errorf("body = %q, want {}", gotBody)
	}

	headers := map[string]string{
		"Accept":          "application/json",
		"Content-Type":    "application/json",
		"Accept-Language": "en",
	}
	for name, want := range headers {
		if value := got.Header.Get(name); value != want {
			t.Errorf("%s = %q, want %q", name, value, want)
		}
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-bogoffers/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-bogoffers/<version> (+<repository>)", agent)
	}
}

// BOGOFFER-SC-001: A query with no language sends no Accept-Language header.
func TestOffersSendsNoLanguage(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`{"code":0,"result":{"offers":[]}}`))
	}))
	t.Cleanup(server.Close)

	_, err := bogoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), bogoffers.OffersQuery{PageSize: 100})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if _, ok := got.Header["Accept-Language"]; ok {
		t.Errorf("Accept-Language = %q, want no header", got.Header.Get("Accept-Language"))
	}
}

// BOGOFFER-SC-001: The client decodes a recorded page in English.
func TestOffersDecodesARecordedPage(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/offers-en-0.json", http.StatusOK)

	page, err := bogoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), bogoffers.OffersQuery{Language: "en", PageSize: 100})
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if page.Code != 0 || page.Result.TotalPageCount != 3 || len(page.Result.Offers) != 100 {
		t.Fatalf(
			"page = code %d, %d pages, %d offers; want 0, 3, 100",
			page.Code, page.Result.TotalPageCount, len(page.Result.Offers),
		)
	}

	index := slices.IndexFunc(page.Result.Offers, func(offer bogoffers.Offer) bool {
		return offer.CampaignID == 17409
	})
	if index < 0 {
		t.Fatal("the page holds no offer 17409")
	}

	type digest struct {
		Title, Category, Segments, Benefit string
		Start                              int64
	}

	ronny := page.Result.Offers[index]
	got := digest{ronny.Title, ronny.CategoryDesc, ronny.SegmentTypes, ronny.BenefitName, 0}

	if ronny.StartDate != nil {
		got.Start = *ronny.StartDate
	}

	want := digest{"Ronny's Pizza", "Restaurants", "RETAIL;SOLO;WM", "DISCOUNT", 1780012800000}
	if got != want {
		t.Errorf("offer 17409 = %+v, want %+v", got, want)
	}
}

// BOGOFFER-SC-002: A status other than 200 is an APIError with the first 1 KiB of the body.
func TestOffersReportsAStatus(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/error.json", http.StatusInternalServerError)

	_, err := bogoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), bogoffers.OffersQuery{PageSize: 100})

	apiErr, ok := errors.AsType[*bogoffers.APIError](err)
	if !ok {
		t.Fatalf("Offers() error = %v, want an *APIError", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError ||
		!strings.Contains(apiErr.Body, "UNEXPECTED_ERROR") {
		t.Errorf("APIError = %d, %q", apiErr.StatusCode, apiErr.Body)
	}

	if !errors.Is(err, bogoffers.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(%v, ErrUnexpectedStatus) = false", err)
	}
}

// BOGOFFER-SC-002: A body that is not a page of offers is an error.
func TestOffersReportsABody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html></html>`))
	}))
	t.Cleanup(server.Close)

	_, err := bogoffers.New(vendorhttp.WithBaseURL(server.URL)).
		Offers(t.Context(), bogoffers.OffersQuery{PageSize: 100})
	if err == nil {
		t.Error("Offers() error = nil, want a decode error")
	}
}
