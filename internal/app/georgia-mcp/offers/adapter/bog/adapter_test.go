package bog_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/bog"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/pkg/bogoffers"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata  = "../../../../../../pkg/bogoffers/testdata"
	firstPage = "en/0"
)

func readAt() time.Time {
	return time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
}

// syncBuffer is a log sink that the handler writes from many goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p) //nolint:wrapcheck // A bytes.Buffer never fails.
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func read(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// adapterFor serves one answer for each page number, by the files of testdata.
func adapterFor(
	t *testing.T,
	pages map[string]func() (int, []byte),
	sink *syncBuffer,
) *bog.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer, ok := pages[r.Header.Get("Accept-Language")+"/"+r.URL.Query().Get("pageNumber")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		status, body := answer()
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	client := bogoffers.New(vendorhttp.WithBaseURL(server.URL))

	return bog.NewAdapter(client, readAt, slog.New(slog.NewTextHandler(sink, nil)))
}

func file(t *testing.T, name string) func() (int, []byte) {
	t.Helper()

	body := read(t, name)

	return func() (int, []byte) { return http.StatusOK, body }
}

func englishPages(t *testing.T) map[string]func() (int, []byte) {
	t.Helper()

	return map[string]func() (int, []byte){
		firstPage: file(t, "offers-en-0.json"),
		"en/1":    file(t, "offers-en-1.json"),
		"en/2":    file(t, "offers-en-2.json"),
	}
}

func find(t *testing.T, catalog app.Catalog, id string) domain.Offer {
	t.Helper()

	for _, offer := range catalog.Offers {
		if offer.ID == id {
			return offer
		}
	}

	t.Fatalf("the catalog holds no offer %s", id)

	return domain.Offer{}
}

func date(t *testing.T, text string) kernel.Date {
	t.Helper()

	value, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return value
}

// BOGOFFER-SC-007: The adapter reads the three pages and maps each offer.
func TestOffersMapsTheRecordedPages(t *testing.T) {
	t.Parallel()

	catalog, err := adapterFor(
		t,
		englishPages(t),
		&syncBuffer{},
	).Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if len(catalog.Offers) != 239 || !catalog.ReadAt.Equal(readAt()) || catalog.Stale {
		t.Fatalf("catalog = %d offers, %v, stale %v; want 239, %v, false",
			len(catalog.Offers), catalog.ReadAt, catalog.Stale, readAt())
	}

	got := find(t, catalog, "17409")
	terms := got.Terms
	got.Terms = ""

	want := domain.Offer{
		Bank:           "bog",
		ID:             "17409",
		Title:          "Ronny's Pizza",
		Benefit:        "20% Discount, unlimited",
		Kinds:          []domain.BenefitKind{domain.KindDiscount},
		Categories:     []domain.Category{domain.CategoryDining},
		BankCategories: []string{"Restaurants"},
		Tiers:          []domain.Tier{domain.TierStandard, domain.TierPremium, domain.TierPrivate},
		Cards:          []domain.Card{{Code: "student_card", Name: "Student Card"}},
		Period:         domain.Period{Start: date(t, "2026-05-29"), End: date(t, "2027-05-22")},
		URL:            "https://bankofgeorgia.ge/en/offers-hub/details/17409",
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(kernel.Date{})); diff != "" {
		t.Errorf("offer 17409 mismatch (-want +got):\n%s", diff)
	}

	if terms == "" || strings.Contains(terms, "<p>") {
		t.Errorf("terms = %q, want a text with no <p>", terms)
	}
}

// BOGOFFER-SC-007: The Georgian catalog has the Georgian address of each offer.
func TestOffersInGeorgian(t *testing.T) {
	t.Parallel()

	pages := map[string]func() (int, []byte){"ka/0": file(t, "offers-ka-0.json")}
	pages["ka/1"] = func() (int, []byte) {
		return http.StatusOK, []byte(`{"code":0,"result":{"totalPageCount":1,"offers":[]}}`)
	}

	catalog, err := adapterFor(t, pages, &syncBuffer{}).Offers(t.Context(), kernel.LanguageGeorgian)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if got := find(
		t,
		catalog,
		"17409",
	).URL; got != "https://bankofgeorgia.ge/ka/offers-hub/details/17409" {
		t.Errorf("URL = %q", got)
	}
}

// BOGOFFER-SC-008: The adapter leaves out an offer with no tier or no title, maps an
// unknown code to its default, and warns one time for each unknown code.
func TestOffersMapsTheEdges(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(
		t,
		map[string]func() (int, []byte){firstPage: file(t, "offers-edge.json")},
		sink,
	)

	catalog, err := adapter.Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	got := make([]string, 0, len(catalog.Offers))
	for _, offer := range catalog.Offers {
		got = append(got, offer.ID)
	}

	if diff := cmp.Diff([]string{"16390", "16401", "16419", "16902"}, got); diff != "" {
		t.Fatalf("ids mismatch (-want +got):\n%s", diff)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{
			"category 999",
			find(t, catalog, "16390").Categories,
			[]domain.Category{domain.CategoryOther},
		},
		{
			"GOLD_CARD",
			find(t, catalog, "16401").Cards,
			[]domain.Card{{Code: "gold_card", Name: "GOLD_CARD"}},
		},
		{"null end", find(t, catalog, "16419").Period.End.IsZero(), true},
		{"OTHER", find(t, catalog, "16902").Kinds, []domain.BenefitKind{domain.KindOther}},
	}

	for _, check := range checks {
		if diff := cmp.Diff(check.want, check.got); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", check.name, diff)
		}
	}

	if _, err := adapter.Offers(t.Context(), kernel.LanguageEnglish); err != nil {
		t.Fatalf("second Offers() error = %v", err)
	}

	log := sink.String()
	for _, code := range []string{"code=VIP", "code=999", "code=GOLD_CARD"} {
		if count := strings.Count(log, code); count != 1 {
			t.Errorf("the log holds %q %d times, want 1:\n%s", code, count, log)
		}
	}
}

// BOGOFFER-SC-009: A failed page fails the read.
func TestOffersFailsOnAPage(t *testing.T) {
	t.Parallel()

	pages := englishPages(t)
	pages["en/1"] = func() (int, []byte) { return http.StatusInternalServerError, read(t, "error.json") }

	_, err := adapterFor(t, pages, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Offers() error = %v, want ErrSourceUnavailable", err)
	}
}

// BOGOFFER-SC-010: A page with a code other than 0 fails the read.
func TestOffersFailsOnACode(t *testing.T) {
	t.Parallel()

	pages := map[string]func() (int, []byte){
		firstPage: func() (int, []byte) { return http.StatusOK, []byte(`{"code":15,"result":null}`) },
	}

	_, err := adapterFor(t, pages, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Offers() error = %v, want ErrSourceUnavailable", err)
	}
}

// BOGOFFER-SC-007: Terms returns the terms of the offer it receives, and Cards returns
// the table of the adapter sorted by code.
func TestTermsAndCards(t *testing.T) {
	t.Parallel()

	adapter := adapterFor(t, nil, &syncBuffer{})

	terms, err := adapter.Terms(
		t.Context(),
		domain.Offer{Terms: "the terms"},
		kernel.LanguageEnglish,
	)
	if err != nil || terms.Text != "the terms" || !terms.ReadAt.Equal(readAt()) {
		t.Errorf("Terms() = %+v, %v", terms, err)
	}

	cards := adapter.Cards()
	if len(cards) != 13 || cards[0].Code != "amex" {
		t.Errorf("Cards() = %v, want 13 cards from amex", cards)
	}

	if bank := adapter.Bank(); bank.Code != "bog" || bank.Name != "Bank of Georgia" {
		t.Errorf("Bank() = %+v", bank)
	}
}
