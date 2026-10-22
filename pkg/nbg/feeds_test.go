package nbg_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/nbg"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

// NBGRATE-SC-035: The RSS feed decodes into its channel and items.
func TestRSSDecodesTheFeed(t *testing.T) {
	t.Parallel()

	var path string

	server := serveFile(t, "testdata/en-2026-10-17.xml")
	recording := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		server.Config.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(recording.Close)

	feed, err := nbg.New(vendorhttp.WithBaseURL(recording.URL)).
		RSS(t.Context(), nbg.LanguageEnglish, nbg.RateSetsQuery{Date: date(t, "2026-10-17")})
	if err != nil {
		t.Fatalf("RSS() error = %v", err)
	}

	if path != "/currencies/en/rss/" {
		t.Errorf("path = %q, want /currencies/en/rss/", path)
	}

	if feed.Channel.Title != "RSS NBG Currency Rates" || len(feed.Channel.Items) != 1 {
		t.Fatalf(
			"channel = %q with %d items, want the NBG title and one item",
			feed.Channel.Title,
			len(feed.Channel.Items),
		)
	}

	item := feed.Channel.Items[0]
	if item.Title != "Currency Rates 2026-10-17" || !strings.Contains(item.Description, dollar) {
		t.Errorf("item = %q, want the title of 2026-10-17 and a description with USD", item.Title)
	}
}

// NBGRATE-SC-036: The codes decode, with USD among them.
func TestCodesDecodesTheList(t *testing.T) {
	t.Parallel()

	var path string

	server := serveFile(t, "testdata/codes.json")
	recording := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		server.Config.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(recording.Close)

	codes, err := nbg.New(vendorhttp.WithBaseURL(recording.URL)).Codes(t.Context())
	if err != nil {
		t.Fatalf("Codes() error = %v", err)
	}

	if path != "/currencies/codes" {
		t.Errorf("path = %q, want /currencies/codes", path)
	}

	if len(codes) != 43 ||
		!slices.ContainsFunc(codes, func(c nbg.Code) bool { return c.Code == dollar }) {
		t.Errorf("codes = %d, want 43 with USD", len(codes))
	}
}

// NBGRATE-SC-037: A status that is not a success is an APIError for the feed and the codes.
func TestFeedsReportAFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := nbg.New(vendorhttp.WithBaseURL(server.URL))

	_, rssErr := client.RSS(t.Context(), nbg.LanguageEnglish, nbg.RateSetsQuery{})
	_, codesErr := client.Codes(t.Context())

	for name, err := range map[string]error{"RSS": rssErr, "Codes": codesErr} {
		apiErr, ok := errors.AsType[*nbg.APIError](err)
		if !ok || apiErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("%s() error = %v, want an APIError with 500", name, err)
		}
	}
}
