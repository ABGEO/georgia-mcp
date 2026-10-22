package liberty_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	libertyadapter "github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/liberty"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/pkg/liberty"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata  = "../../../../../../pkg/liberty/testdata"
	emptyList = `{"data":{"list":[]},"succeeded":true,"message":""}`
	keyCats   = "categories"
	keyTypes  = "types"
)

// readAt is 2026-10-20 12:00 in Tbilisi.
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

// fakeLiberty serves the files of testdata by the method and the filter of a request:
// "offers-<page>", "category-<id>-<page>", "type-<id>-<page>", "details-<id>". A list with
// no file is empty. failing keys answer 500.
type fakeLiberty struct {
	bodies  map[string][]byte
	failing []string
}

func recorded(t *testing.T) *fakeLiberty {
	t.Helper()

	names, err := filepath.Glob(filepath.Join(testdata, "*"))
	if err != nil {
		t.Fatal(err)
	}

	fake := &fakeLiberty{bodies: map[string][]byte{}}

	for _, name := range names {
		body, err := os.ReadFile(name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		base := filepath.Base(name)
		fake.bodies[strings.TrimSuffix(strings.TrimSuffix(base, ".json"), ".html")] = body
	}

	return fake
}

func (f *fakeLiberty) key(r *http.Request) string {
	switch path.Base(r.URL.Path) {
	case "loadcategories":
		return keyCats
	case "loadtypes":
		return keyTypes
	case "loadOffers":
		var body map[string]any

		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)

		page, _ := body["page"].(float64)

		if category, ok := body["categoryId"].(float64); ok {
			return fmt.Sprintf("category-%d-%d", int(category), int(page))
		}

		if offerType, ok := body["typeId"].(float64); ok {
			return fmt.Sprintf("type-%d-%d", int(offerType), int(page))
		}

		return fmt.Sprintf("offers-%d", int(page))
	default:
		return "details-" + path.Base(r.URL.Path)
	}
}

func (f *fakeLiberty) adapter(t *testing.T, sink *syncBuffer) *libertyadapter.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := f.key(r)
		if slices.ContainsFunc(
			f.failing,
			func(prefix string) bool { return strings.HasPrefix(key, prefix) },
		) {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		body, ok := f.bodies[key]
		if !ok && !strings.HasPrefix(key, "details-") {
			body, ok = []byte(emptyList), true
		}

		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	client := liberty.New(vendorhttp.WithBaseURL(server.URL))

	return libertyadapter.NewAdapter(client, readAt, slog.New(slog.NewTextHandler(sink, nil)))
}

func find(t *testing.T, catalog app.Catalog, id string) domain.Offer {
	t.Helper()

	index := slices.IndexFunc(
		catalog.Offers,
		func(offer domain.Offer) bool { return offer.ID == id },
	)
	if index < 0 {
		t.Fatalf("the catalog holds no offer %s", id)
	}

	return catalog.Offers[index]
}

// idsOf returns the ids of the offers of the files of one filter.
func idsOf(t *testing.T, fake *fakeLiberty, prefix string) []string {
	t.Helper()

	var ids []string

	for key, body := range fake.bodies {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		var page struct {
			Data struct {
				List []liberty.Offer `json:"list"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			t.Fatal(err)
		}

		for _, offer := range page.Data.List {
			ids = append(ids, strconv.Itoa(offer.ID))
		}
	}

	return ids
}

func date(t *testing.T, text string) kernel.Date {
	t.Helper()

	value, err := kernel.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}

	return value
}

// LIBOFFER-SC-004: The catalog holds the 206 offers, each of the tier premium. Offer 727
// has its title, its label as the benefit, no start date, and the end of its days left.
func TestOffersMapTheRecordedPages(t *testing.T) {
	t.Parallel()

	catalog, err := recorded(
		t,
	).adapter(t, &syncBuffer{}).
		Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if len(catalog.Offers) != 206 || !catalog.ReadAt.Equal(readAt()) {
		t.Fatalf(
			"catalog = %d offers at %v; want 206 at %v",
			len(catalog.Offers),
			catalog.ReadAt,
			readAt(),
		)
	}

	for _, offer := range catalog.Offers {
		if !slices.Equal(offer.Tiers, []domain.Tier{domain.TierPremium}) {
			t.Errorf("offer %s tiers = %v, want [premium]", offer.ID, offer.Tiers)
		}
	}

	got := find(t, catalog, "727")

	want := domain.Offer{
		Bank:           "liberty",
		ID:             "727",
		Title:          "Hyatt Regency Bangkok Sukhumvit",
		Benefit:        "Special Hotel Offer",
		Kinds:          got.Kinds,
		Categories:     got.Categories,
		BankCategories: got.BankCategories,
		Tiers:          []domain.Tier{domain.TierPremium},
		Cards:          got.Cards,
		Period:         domain.Period{End: date(t, "2026-12-31")},
		URL:            "https://studioliberty.ge/en/studio-lifestyle/offers/details/727",
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(kernel.Date{})); diff != "" {
		t.Errorf("offer 727 mismatch (-want +got):\n%s", diff)
	}
}

// LIBOFFER-SC-005: Each offer gets the category, the kind, and the card of each list that
// holds it. An offer in no type has the kind other and no card.
func TestOffersJoinTheLists(t *testing.T) {
	t.Parallel()

	fake := recorded(t)

	catalog, err := fake.adapter(t, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	for _, id := range idsOf(t, fake, "category-6-") {
		offer := find(t, catalog, id)
		if !slices.Contains(offer.Categories, domain.CategoryLeisure) ||
			!slices.Contains(offer.BankCategories, "Leisure") {
			t.Errorf("leisure offer %s = %v %v", id, offer.Categories, offer.BankCategories)
		}
	}

	for _, id := range idsOf(t, fake, "type-2-") {
		if kinds := find(t, catalog, id).Kinds; !slices.Contains(kinds, domain.KindCashback) {
			t.Errorf("cashback offer %s kinds = %v", id, kinds)
		}
	}

	for _, id := range idsOf(t, fake, "type-3-") {
		if cards := find(
			t,
			catalog,
			id,
		).Cards; !slices.Contains(
			cards,
			domain.Card{Code: "visa", Name: "Visa"},
		) {
			t.Errorf("Visa offer %s cards = %v", id, cards)
		}
	}
}

// LIBOFFER-SC-005: An offer in no type has the kind other and no card.
func TestOffersInNoType(t *testing.T) {
	t.Parallel()

	fake := recorded(t)

	catalog, err := fake.adapter(t, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	typed := slices.Concat(idsOf(t, fake, "type-2-"), idsOf(t, fake, "type-3-"))

	lonely := slices.IndexFunc(
		catalog.Offers,
		func(offer domain.Offer) bool { return !slices.Contains(typed, offer.ID) },
	)
	if lonely < 0 {
		t.Fatal("every offer is in a type")
	}

	if offer := catalog.Offers[lonely]; !slices.Equal(
		offer.Kinds,
		[]domain.BenefitKind{domain.KindOther},
	) ||
		len(offer.Cards) != 0 {
		t.Errorf("offer in no type = %v %v, want [other] and no card", offer.Kinds, offer.Cards)
	}
}

// LIBOFFER-SC-006: The days left give the end date after the date of the read. A text with
// no number gives no end date.
func TestOffersReadTheDaysLeft(t *testing.T) {
	t.Parallel()

	fake := recorded(t)
	fake.bodies = map[string][]byte{
		keyCats:  fake.bodies[keyCats],
		keyTypes: fake.bodies[keyTypes],
		"offers-1": []byte(`{"succeeded":true,"data":{"list":[` +
			`{"id":1,"title":"A","labels":["5% Cashback"],"remainingDays":"72 Days left"},` +
			`{"id":2,"title":"B","labels":["5% Cashback"],"remainingDays":"დარჩა 3 დღე"},` +
			`{"id":3,"title":"C","labels":["5% Cashback"],"remainingDays":"Last day"}]}}`),
	}

	catalog, err := fake.adapter(t, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	got := []string{
		find(t, catalog, "1").Period.End.String(),
		find(t, catalog, "2").Period.End.String(),
		strconv.FormatBool(find(t, catalog, "3").Period.End.IsZero()),
	}

	if diff := cmp.Diff([]string{"2026-12-31", "2026-10-23", "true"}, got); diff != "" {
		t.Errorf("end dates mismatch (-want +got):\n%s", diff)
	}
}

// LIBOFFER-SC-007: The terms are the paragraphs of the page of the offer, one line each,
// with no blank line.
func TestTermsReadThePage(t *testing.T) {
	t.Parallel()

	adapter := recorded(t).adapter(t, &syncBuffer{})

	terms, err := adapter.Terms(t.Context(), domain.Offer{ID: "727"}, kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Terms() error = %v", err)
	}

	lines := strings.Split(terms.Text, "\n")
	if len(lines) != 13 || lines[12] != "• The offer is valid until December 31, 2026" ||
		!terms.ReadAt.Equal(readAt()) {
		t.Errorf("terms = %d lines, last %q, at %v", len(lines), lines[len(lines)-1], terms.ReadAt)
	}

	if _, err := adapter.Terms(
		t.Context(),
		domain.Offer{ID: "1"},
		kernel.LanguageEnglish,
	); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("Terms(1) error = %v, want ErrSourceUnavailable", err)
	}
}

// LIBOFFER-SC-009: A failed list fails the read.
func TestOffersFailOnAType(t *testing.T) {
	t.Parallel()

	fake := recorded(t)
	fake.failing = []string{"type-2-"}

	_, err := fake.adapter(t, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Offers() error = %v, want ErrSourceUnavailable", err)
	}
}

// LIBOFFER-SC-005: The bank is Liberty Bank, and its cards are the three cards of the types.
func TestBankAndCards(t *testing.T) {
	t.Parallel()

	adapter := recorded(t).adapter(t, &syncBuffer{})

	if bank := adapter.Bank(); bank != (domain.Bank{Code: "liberty", Name: "Liberty Bank"}) {
		t.Errorf("Bank() = %+v", bank)
	}

	want := []domain.Card{
		{Code: "mastercard", Name: "Mastercard"},
		{Code: "top_card_black", Name: "Top|Card Black Edition"},
		{Code: "visa", Name: "Visa"},
	}
	if diff := cmp.Diff(want, adapter.Cards()); diff != "" {
		t.Errorf("Cards() mismatch (-want +got):\n%s", diff)
	}
}

// LIBOFFER-SC-010: An offer with no title takes the words of the slug of its address. An
// offer with no title and no slug is absent.
func TestOffersTitleFromTheSlug(t *testing.T) {
	t.Parallel()

	fake := recorded(t)
	fake.bodies = map[string][]byte{
		keyCats:  fake.bodies[keyCats],
		keyTypes: fake.bodies[keyTypes],
		"offers-1": []byte(`{"succeeded":true,"data":{"list":[` +
			`{"id":724,"title":"","labels":["Special Offers"],` +
			`"url":"https://StudioLiberty.ge/en/studio-lifestyle/offers/details/724-the-westin-grande-sukhumvit "},` +
			`{"id":725,"title":"","labels":["Special Offers"],` +
			`"url":"https://StudioLiberty.ge/en/studio-lifestyle/offers/details/725"},` +
			`{"id":727,"title":"Hyatt","labels":["Special Offers"],` +
			`"url":"https://StudioLiberty.ge/en/studio-lifestyle/offers/details/727-hyatt"}]}}`),
	}

	catalog, err := fake.adapter(t, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	titles := map[string]string{}
	for _, offer := range catalog.Offers {
		titles[offer.ID] = offer.Title
	}

	want := map[string]string{"724": "The Westin Grande Sukhumvit", "727": "Hyatt"}
	if diff := cmp.Diff(want, titles); diff != "" {
		t.Errorf("titles mismatch (-want +got):\n%s", diff)
	}
}
