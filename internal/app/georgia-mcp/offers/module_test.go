package offers_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers"
	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
	"github.com/abgeo/georgia-mcp/internal/pkg/mcpserver"
)

const (
	testdata     = "../../../../pkg/bogoffers/testdata"
	searchOffers = "offers_search_offers"
	getOffer     = "offers_get_offer"
	bogCode      = "bog"
	argBanks     = "banks"
)

type offerJSON struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Partner   string   `json:"partner"`
	Tiers     []string `json:"tiers"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	URL       string   `json:"url"`
}

type searchJSON struct {
	Banks []struct {
		Bank       string      `json:"bank"`
		Source     string      `json:"source"`
		AsOf       string      `json:"as_of"`
		Total      int         `json:"total"`
		NextOffset int         `json:"next_offset"`
		Offers     []offerJSON `json:"offers"`
	} `json:"banks"`
}

type detailJSON struct {
	Offer offerJSON `json:"offer"`
	Terms string    `json:"terms"`
}

// bogServer serves the recorded pages of BOG by language and page number, and counts
// the calls of each language.
type bogServer struct {
	calls sync.Map
	url   url.URL
}

func newBOGServer(t *testing.T) *bogServer {
	t.Helper()

	pages := map[string][]byte{}

	for _, name := range []string{"offers-en-0.json", "offers-en-1.json", "offers-en-2.json", "offers-ka-0.json"} {
		path := filepath.Join(testdata, name)

		body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		pages[strings.TrimSuffix(strings.TrimPrefix(name, "offers-"), ".json")] = body
	}

	bog := &bogServer{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		language := r.Header.Get("Accept-Language")
		counter, _ := bog.calls.LoadOrStore(language, &atomic.Int32{})
		counter.(*atomic.Int32).Add(1) //nolint:forcetypeassert // The map holds counters only.

		body, ok := pages[language+"-"+r.URL.Query().Get("pageNumber")]
		if !ok && language == "ka" {
			body, ok = pages["en-"+r.URL.Query().Get("pageNumber")]
		}

		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	bog.url = *base

	return bog
}

func (b *bogServer) count(language string) int32 {
	counter, ok := b.calls.Load(language)
	if !ok {
		return 0
	}

	return counter.(*atomic.Int32).Load() //nolint:forcetypeassert // The map holds counters only.
}

func config(bog *bogServer) offers.Config {
	return offers.Config{BOG: offers.BOGConfig{
		Enabled: true,
		BaseURL: bog.url,
		HTTP:    offers.BOGHTTPConfig{Timeout: 10 * time.Second, Rate: 100, Burst: 100},
	}}
}

// now is 2026-10-20 12:00 in Tbilisi.
func now() time.Time {
	return time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
}

func serve(t *testing.T, cfg offers.Config) *mcp.ClientSession {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)

	module, err := offers.NewWithClock(cfg, logger, now)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = module.Close() })

	registry := mcpserver.New(buildinfo.Info{Version: "v0"}, logger)
	if err := registry.Register(module); err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() { _ = registry.RunStdio(ctx, serverTransport) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).
		Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = session.Close() })

	return session
}

func call(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) *mcp.CallToolResult {
	t.Helper()

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Errorf("CallTool(%s) error = %v", name, err)

		return &mcp.CallToolResult{IsError: true}
	}

	return result
}

func decode(t *testing.T, result *mcp.CallToolResult, out any) {
	t.Helper()

	if result.IsError {
		t.Fatalf("result is an error: %+v", result.Content)
	}

	body, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(body, out); err != nil {
		t.Fatal(err)
	}
}

// BOGOFFER-SC-022: A client searches the leisure offers of SOLO at BOG, reads the terms
// of the first, and gets a message for an unknown tier.
func TestSearchAndGetAnOfferOfBOG(t *testing.T) {
	t.Parallel()

	session := serve(t, config(newBOGServer(t)))

	var found searchJSON

	decode(t, call(t, session, searchOffers, map[string]any{
		argBanks: []string{
			bogCode,
		},
		"tiers":      []string{"premium"},
		"categories": []string{"leisure"},
		"limit":      5,
	}), &found)

	if len(found.Banks) != 1 || found.Banks[0].Source != "Bank of Georgia" ||
		found.Banks[0].AsOf == "" {
		t.Fatalf("banks = %+v, want Bank of Georgia with as_of", found.Banks)
	}

	bank := found.Banks[0]
	checkPremiumOffers(t, bank.Offers, bank.Total)

	var detail detailJSON

	decode(
		t,
		call(t, session, getOffer, map[string]any{"bank": bogCode, "id": bank.Offers[0].ID}),
		&detail,
	)

	if detail.Offer.ID != bank.Offers[0].ID || detail.Terms == "" {
		t.Errorf("get = %+v, want the terms of %s", detail, bank.Offers[0].ID)
	}

	result := call(t, session, searchOffers, map[string]any{"tiers": []string{"vip"}})
	if !result.IsError {
		t.Error("search with the tier vip succeeded, want a tool error")
	}
}

func checkPremiumOffers(t *testing.T, offers []offerJSON, total int) {
	t.Helper()

	if len(offers) == 0 || len(offers) > 5 || total < len(offers) {
		t.Fatalf("offers = %d of %d, want 1 to 5", len(offers), total)
	}

	for _, offer := range offers {
		if !slices.Contains(offer.Tiers, "premium") ||
			!strings.HasPrefix(offer.URL, "https://bankofgeorgia.ge/en/offers-hub/details/") {
			t.Errorf("offer %s = %+v, want premium and a BOG url", offer.ID, offer)
		}
	}
}

// BOGOFFER-SC-019: 20 searches at once in en send 3 calls. One search in ka sends 3 more.
func TestSearchesShareOneRead(t *testing.T) {
	t.Parallel()

	bog := newBOGServer(t)
	session := serve(t, config(bog))

	var wg sync.WaitGroup

	for range 20 {
		wg.Go(func() { call(t, session, searchOffers, map[string]any{}) })
	}

	wg.Wait()

	call(t, session, searchOffers, map[string]any{"language": "ka"})

	if en, ka := bog.count("en"), bog.count("ka"); en != 3 || ka != 3 {
		t.Errorf("calls = en %d, ka %d; want 3 and 3", en, ka)
	}
}

// BOGOFFER-SC-020: With the catalog in the cache, each search answers in less than 500 ms.
func TestSearchAnswersFromTheCache(t *testing.T) {
	t.Parallel()

	session := serve(t, config(newBOGServer(t)))
	call(t, session, searchOffers, map[string]any{})

	var slowest time.Duration

	for range 100 {
		start := time.Now()

		call(t, session, searchOffers, map[string]any{"query": "a"})

		slowest = max(slowest, time.Since(start))
	}

	if slowest >= 500*time.Millisecond {
		t.Errorf("slowest search = %v, want less than 500ms", slowest)
	}
}
