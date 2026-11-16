package tbc_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/tbc"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	"github.com/abgeo/georgia-mcp/pkg/tbcoffers"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata  = "../../../../../../pkg/tbcoffers/testdata"
	oktopusID = "7tD3DL0YuO2V6f1Efr4sSV"
	emptyList = `{"pagingDetails":{"totalCount":0,"hasNextPage":false},"list":[]}`
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

// fakeTBC serves the recorded answers. bodies maps "filters/<segment>", "list/<filter>",
// and "offer/<slug>" to an answer. A list with no answer is empty. failing lists answer 500.
type fakeTBC struct {
	bodies  map[string][]byte
	failing []string
}

func recorded(t *testing.T) *fakeTBC {
	t.Helper()

	return &fakeTBC{bodies: map[string][]byte{
		"filters/Concept":          read(t, "filters-concept.json"),
		"filters/ForEveryone":      read(t, "filters-foreveryone.json"),
		"filters/ForYouth":         read(t, "filters-foryouth.json"),
		"list/":                    read(t, "offers.json"),
		"list/Category:Leisure":    read(t, "filter-category-leisure.json"),
		"list/OfferType:Cashback":  read(t, "filter-offertype-cashback.json"),
		"list/CardType:CreditCard": read(t, "filter-cardtype-creditcard.json"),
		"offer/oktopusi-qeshbeqi":  read(t, "offer-oktopusi-qeshbeqi.json"),
	}}
}

func (f *fakeTBC) key(r *http.Request) string {
	switch r.URL.Path {
	case "/filters":
		return "filters/" + r.URL.Query().Get("segment")
	case "/entries/offer":
		var request tbcoffers.OffersRequest

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &request)

		return "list/" + strings.Join(request.Filter, ",")
	default:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

		return "offer/" + parts[len(parts)-2]
	}
}

func (f *fakeTBC) adapter(t *testing.T, sink *syncBuffer) *tbc.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := f.key(r)
		if slices.Contains(f.failing, key) {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		body, ok := f.bodies[key]
		if !ok && strings.HasPrefix(key, "list/") {
			body, ok = []byte(emptyList), true
		}

		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	client := tbcoffers.New(vendorhttp.WithBaseURL(server.URL))

	return tbc.NewAdapter(client, readAt, slog.New(slog.NewTextHandler(sink, nil)))
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	return body
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

func idsOf(t *testing.T, name string) []string {
	t.Helper()

	var page tbcoffers.OffersPage
	if err := json.Unmarshal(read(t, name), &page); err != nil {
		t.Fatal(err)
	}

	ids := make([]string, 0, len(page.List))
	for _, entry := range page.List {
		ids = append(ids, entry.ID)
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

// TBCOFFER-SC-005: Each offer gets the values of each filter that lists it. An offer in
// no filter has the category other and the kind other.
func TestOffersJoinTheFilters(t *testing.T) {
	t.Parallel()

	catalog, err := recorded(
		t,
	).adapter(t, &syncBuffer{}).
		Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	if len(catalog.Offers) != 517 || !catalog.ReadAt.Equal(readAt()) {
		t.Errorf(
			"catalog = %d offers at %v; want 517 at %v",
			len(catalog.Offers),
			catalog.ReadAt,
			readAt(),
		)
	}

	for _, id := range idsOf(t, "filter-category-leisure.json") {
		offer := find(t, catalog, id)
		if !slices.Contains(offer.Categories, domain.CategoryLeisure) ||
			!slices.Contains(offer.BankCategories, "Leisure") {
			t.Errorf("leisure offer %s = %v %v", id, offer.Categories, offer.BankCategories)
		}
	}

	for _, id := range idsOf(t, "filter-cardtype-creditcard.json") {
		if offer := find(
			t,
			catalog,
			id,
		); !slices.Contains(
			offer.Cards,
			domain.Card{Code: "visa", Name: "Visa"},
		) {
			t.Errorf("Visa offer %s has the cards %v", id, offer.Cards)
		}
	}

	if kinds := find(
		t,
		catalog,
		oktopusID,
	).Kinds; !slices.Equal(
		kinds,
		[]domain.BenefitKind{domain.KindCashback},
	) {
		t.Errorf("Oktopus kinds = %v, want [cashback]", kinds)
	}
}

// TBCOFFER-SC-005: An offer that no filter lists has the category other and the kind other.
func TestOffersInNoFilter(t *testing.T) {
	t.Parallel()

	catalog, err := recorded(
		t,
	).adapter(t, &syncBuffer{}).
		Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	listed := slices.Concat(
		idsOf(t, "filter-category-leisure.json"),
		idsOf(t, "filter-offertype-cashback.json"),
		idsOf(t, "filter-cardtype-creditcard.json"),
	)

	lonely := slices.IndexFunc(catalog.Offers, func(offer domain.Offer) bool {
		return !slices.Contains(listed, offer.ID)
	})
	if lonely < 0 {
		t.Fatal("no offer is in no category filter")
	}

	if offer := catalog.Offers[lonely]; !slices.Equal(
		offer.Categories,
		[]domain.Category{domain.CategoryOther},
	) ||
		!slices.Equal(offer.Kinds, []domain.BenefitKind{domain.KindOther}) {
		t.Errorf("offer in no filter = %v %v, want [other] [other]", offer.Categories, offer.Kinds)
	}
}

// TBCOFFER-SC-004: An offer has its title, its partner, its description, its period, its
// tiers by the id of the segment, and the address of its site.
func TestOffersMapAnEntry(t *testing.T) {
	t.Parallel()

	catalog, err := recorded(
		t,
	).adapter(t, &syncBuffer{}).
		Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	got := find(t, catalog, oktopusID)

	want := domain.Offer{
		Bank:           "tbc",
		ID:             oktopusID,
		Title:          "Oktopus",
		Partner:        "Oktopus Fitness Club",
		Benefit:        got.Benefit,
		Kinds:          []domain.BenefitKind{domain.KindCashback},
		Categories:     got.Categories,
		BankCategories: got.BankCategories,
		Tiers:          []domain.Tier{domain.TierPremium},
		Cards:          got.Cards,
		Period:         domain.Period{Start: date(t, "2026-10-20"), End: date(t, "2026-10-21")},
		URL:            "https://tbcconcept.ge/en/offers/oktopusi-qeshbeqi",
	}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(kernel.Date{})); diff != "" {
		t.Errorf("Oktopus mismatch (-want +got):\n%s", diff)
	}

	if !strings.HasPrefix(got.Benefit, "As part of the TBC Concept offer") || got.Terms != "" {
		t.Errorf("benefit = %q, terms = %q", got.Benefit, got.Terms)
	}

	checkTheRecordedTiers(t, catalog)
}

// checkTheRecordedTiers checks that the offer of Installment is absent, that the offer
// of Emigrants has the tier standard, and that an offer with no premium tier has its
// address on tbcbank.ge.
func checkTheRecordedTiers(t *testing.T, catalog app.Catalog) {
	t.Helper()

	const (
		installment = "12-Month Interest-Free Installment Plan, Effective 4.3%"
		emigrants   = "A Special Offer for Emigrants from TBC Bank and Metropol"
	)

	foundEmigrants := false

	for _, offer := range catalog.Offers {
		switch offer.Title {
		case installment:
			t.Errorf("the catalog holds %q, an offer with no tier", offer.Title)
		case emigrants:
			foundEmigrants = true

			if !slices.Equal(offer.Tiers, []domain.Tier{domain.TierStandard}) {
				t.Errorf("the offer of Emigrants has the tiers %v, want [standard]", offer.Tiers)
			}
		}

		if !slices.Contains(offer.Tiers, domain.TierPremium) &&
			!strings.HasPrefix(offer.URL, "https://tbcbank.ge/en/offers/all-offers/") {
			t.Errorf("offer %s has the address %s", offer.ID, offer.URL)
		}
	}

	if !foundEmigrants {
		t.Errorf("the catalog holds no %q", emigrants)
	}
}

// TBCOFFER-SC-006: An entry with a new segment only is absent, a new category gives
// other, and the log holds one warning for each new code.
func TestOffersMapTheUnknownCodes(t *testing.T) {
	t.Parallel()

	fake := recorded(t)

	var concept []tbcoffers.Filter
	if err := json.Unmarshal(fake.bodies["filters/Concept"], &concept); err != nil {
		t.Fatal(err)
	}

	concept[0].Childs = append(concept[0].Childs, tbcoffers.Filter{Code: "Pets", Name: "Pets"})
	fake.bodies["filters/Concept"] = mustMarshal(t, concept)
	fake.bodies["list/Category:Pets"] = []byte(
		`{"pagingDetails":{"totalCount":1},"list":[{"$id":"` + oktopusID + `"}]}`,
	)

	var page tbcoffers.OffersPage
	if err := json.Unmarshal(fake.bodies["list/"], &page); err != nil {
		t.Fatal(err)
	}

	page.List = append(page.List, tbcoffers.Entry{
		ID: "newSegmentOnly", Title: "New", Slug: "new",
		Segments: []tbcoffers.Segment{{ID: "brandNewSegment", Label: "New"}},
	})
	fake.bodies["list/"] = mustMarshal(t, page)

	sink := &syncBuffer{}
	adapter := fake.adapter(t, sink)

	for range 2 {
		catalog, err := adapter.Offers(t.Context(), kernel.LanguageEnglish)
		if err != nil {
			t.Fatalf("Offers() error = %v", err)
		}

		if slices.ContainsFunc(
			catalog.Offers,
			func(offer domain.Offer) bool { return offer.ID == "newSegmentOnly" },
		) {
			t.Error("the catalog holds the offer of the new segment")
		}

		if categories := find(
			t,
			catalog,
			oktopusID,
		).Categories; !slices.Contains(
			categories,
			domain.CategoryOther,
		) {
			t.Errorf("Oktopus categories = %v, want other for Pets", categories)
		}
	}

	log := sink.String()
	for _, code := range []string{"code=brandNewSegment", "code=Pets"} {
		if count := strings.Count(log, code); count != 1 {
			t.Errorf("the log holds %q %d times, want 1:\n%s", code, count, log)
		}
	}
}

// entryOf returns an entry whose id, title, and slug are id, with the segments.
func entryOf(id string, segments ...string) tbcoffers.Entry {
	list := make([]tbcoffers.Segment, 0, len(segments))
	for _, segment := range segments {
		list = append(list, tbcoffers.Segment{ID: segment})
	}

	return tbcoffers.Entry{ID: id, Title: id, Slug: id, Segments: list}
}

// addEntries adds the entries to the recorded list of every offer.
func addEntries(t *testing.T, fake *fakeTBC, entries ...tbcoffers.Entry) {
	t.Helper()

	var page tbcoffers.OffersPage
	if err := json.Unmarshal(fake.bodies["list/"], &page); err != nil {
		t.Fatal(err)
	}

	page.List = append(page.List, entries...)
	fake.bodies["list/"] = mustMarshal(t, page)
}

// TBCOFFER-SC-011: Emigrants is standard, Concept Partners is premium, and Expats and
// Installment give no tier with no warning.
func TestOffersMapTheSegments(t *testing.T) {
	t.Parallel()

	const (
		emigrants       = "6gyl5FRPnrS66J79o230lD"
		conceptPartners = "4QFNANlkPnt4tuCYva9nN9"
		installment     = "5FSNwGngwsxY6Giq6U8Ani"
		concept         = "5I27EbKG2HykRaU760nvck"
		expats          = "6WK1EC8bbUzUG02i51urvP"
	)

	fake := recorded(t)
	addEntries(t, fake,
		entryOf("emigrants", emigrants),
		entryOf("partners", conceptPartners),
		entryOf("installment", installment),
		entryOf("conceptExpats", concept, expats),
	)

	sink := &syncBuffer{}

	catalog, err := fake.adapter(t, sink).Offers(t.Context(), kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Offers() error = %v", err)
	}

	want := map[string][]domain.Tier{
		"emigrants":     {domain.TierStandard},
		"partners":      {domain.TierPremium},
		"conceptExpats": {domain.TierPremium},
	}
	for id, tiers := range want {
		if got := find(t, catalog, id).Tiers; !slices.Equal(got, tiers) {
			t.Errorf("%s tiers = %v, want %v", id, got, tiers)
		}
	}

	if address := find(t, catalog, "partners").URL; !strings.Contains(address, "tbcconcept.ge") {
		t.Errorf("partners URL = %s, want an address on tbcconcept.ge", address)
	}

	if slices.ContainsFunc(
		catalog.Offers,
		func(offer domain.Offer) bool { return offer.ID == "installment" },
	) {
		t.Error("the catalog holds the offer of Installment")
	}

	if log := sink.String(); strings.Contains(log, "field=segment") {
		t.Errorf("the log warns of a segment:\n%s", log)
	}
}

// TBCOFFER-SC-008: A failed filter list fails the read.
func TestOffersFailOnAFilter(t *testing.T) {
	t.Parallel()

	fake := recorded(t)
	fake.failing = []string{"list/OfferType:Cashback"}

	_, err := fake.adapter(t, &syncBuffer{}).Offers(t.Context(), kernel.LanguageEnglish)
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("Offers() error = %v, want ErrSourceUnavailable", err)
	}
}

// TBCOFFER-SC-007: The terms come from the page of the offer: one line for each
// paragraph and list item, a list item with "- ", and no blank line.
func TestTermsReadThePage(t *testing.T) {
	t.Parallel()

	adapter := recorded(t).adapter(t, &syncBuffer{})
	offer := domain.Offer{ID: oktopusID, URL: "https://tbcconcept.ge/en/offers/oktopusi-qeshbeqi"}

	terms, err := adapter.Terms(t.Context(), offer, kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Terms() error = %v", err)
	}

	lines := strings.Split(terms.Text, "\n")
	if !slices.Contains(lines, "- The offer applies only to 6-month memberships.") ||
		slices.Contains(
			lines,
			"",
		) || !strings.HasPrefix(lines[0], "As part of the TBC Concept offer") {
		t.Errorf("terms = %q", terms.Text)
	}

	if !terms.ReadAt.Equal(readAt()) {
		t.Errorf("ReadAt = %v, want %v", terms.ReadAt, readAt())
	}

	if _, err := adapter.Terms(
		t.Context(),
		domain.Offer{URL: "https://tbcbank.ge/en/offers/all-offers/gone"},
		kernel.LanguageEnglish,
	); !errors.Is(
		err,
		kernel.ErrSourceUnavailable,
	) {
		t.Errorf("Terms(gone) error = %v, want ErrSourceUnavailable", err)
	}
}

// TBCOFFER-SC-005: The bank is TBC Bank, and its cards are the tables of the adapter.
func TestBankAndCards(t *testing.T) {
	t.Parallel()

	adapter := recorded(t).adapter(t, &syncBuffer{})

	if bank := adapter.Bank(); bank.Code != "tbc" || bank.Name != "TBC Bank" {
		t.Errorf("Bank() = %+v", bank)
	}

	cards := adapter.Cards()
	if len(cards) != 11 ||
		!slices.IsSortedFunc(
			cards,
			func(a, b domain.Card) int { return strings.Compare(string(a.Code), string(b.Code)) },
		) {
		t.Errorf("Cards() = %v, want 11 sorted by code", cards)
	}
}
