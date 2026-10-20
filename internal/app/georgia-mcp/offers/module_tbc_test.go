package offers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers"
)

const (
	tbcTestdata = "../../../../pkg/tbcoffers/testdata"
	tbcCode     = "tbc"
	oktopusID   = "7tD3DL0YuO2V6f1Efr4sSV"
)

// tbcServer serves the recorded answers of TBC. A filter with no recorded answer is an
// empty list.
func tbcServer(t *testing.T) url.URL {
	t.Helper()

	files := map[string]string{
		"filters/Concept":          "filters-concept.json",
		"filters/ForEveryone":      "filters-foreveryone.json",
		"filters/ForYouth":         "filters-foryouth.json",
		"list/":                    "offers.json",
		"list/Category:Leisure":    "filter-category-leisure.json",
		"list/OfferType:Cashback":  "filter-offertype-cashback.json",
		"list/CardType:CreditCard": "filter-cardtype-creditcard.json",
		"offer/oktopusi-qeshbeqi":  "offer-oktopusi-qeshbeqi.json",
	}

	bodies := make(map[string][]byte, len(files))

	for key, name := range files {
		path := filepath.Join(tbcTestdata, name)

		body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		bodies[key] = body
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[tbcKey(r)]
		if !ok && strings.HasPrefix(tbcKey(r), "list/") {
			body, ok = []byte(`{"pagingDetails":{"totalCount":0},"list":[]}`), true
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

	return *base
}

func tbcKey(r *http.Request) string {
	switch r.URL.Path {
	case "/filters":
		return "filters/" + r.URL.Query().Get("segment")
	case "/entries/offer":
		var request struct {
			Filter []string `json:"filter"`
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &request)
		r.Body = io.NopCloser(strings.NewReader(string(body)))

		return "list/" + strings.Join(request.Filter, ",")
	default:
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

		return "offer/" + parts[len(parts)-2]
	}
}

func tbcConfig(t *testing.T) offers.Config {
	t.Helper()

	return offers.Config{TBC: offers.TBCConfig{
		Enabled: true,
		BaseURL: tbcServer(t),
		HTTP:    offers.TBCHTTPConfig{Timeout: 10 * time.Second, Rate: 100, Burst: 100},
	}}
}

// allOffers reads every page of the offers of one bank with the offset.
func allOffers(t *testing.T, session *mcp.ClientSession, bank string) []offerJSON {
	t.Helper()

	var collected []offerJSON

	for offset := 0; ; {
		var found searchJSON

		decode(t, call(t, session, searchOffers, map[string]any{
			argBanks: []string{bank}, "limit": 30, "offset": offset,
		}), &found)

		collected = append(collected, found.Banks[0].Offers...)

		if found.Banks[0].NextOffset == 0 {
			return collected
		}

		offset = found.Banks[0].NextOffset
	}
}

func textOf(result *mcp.CallToolResult) string {
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return ""
	}

	return content.Text
}

// TBCOFFER-SC-004: The search of TBC gives Oktopus with its tier, partner, period, and
// the address on the site of Concept. No offer ended before today, and no offer of
// Emigrants or Installment only is in the answer.
func TestSearchOfTBC(t *testing.T) {
	t.Parallel()

	collected := allOffers(t, serve(t, tbcConfig(t)), tbcCode)

	index := slices.IndexFunc(
		collected,
		func(offer offerJSON) bool { return offer.ID == oktopusID },
	)
	if index < 0 {
		t.Fatal("the answer holds no Oktopus")
	}

	want := offerJSON{
		ID:        oktopusID,
		Title:     "Oktopus",
		Partner:   "Oktopus Fitness Club",
		Tiers:     []string{"premium"},
		StartDate: "2026-10-20",
		EndDate:   "2026-10-21",
		URL:       "https://tbcconcept.ge/en/offers/oktopusi-qeshbeqi",
	}
	if got := collected[index]; !equalOffer(got, want) {
		t.Errorf("Oktopus = %+v, want %+v", got, want)
	}

	tierless := []string{
		"A Special Offer for Emigrants from TBC Bank and Metropol",
		"12-Month Interest-Free Installment Plan, Effective 4.3%",
	}

	for _, offer := range collected {
		if offer.EndDate != "" && offer.EndDate < "2026-10-20" {
			t.Errorf("offer %s ended on %s", offer.ID, offer.EndDate)
		}

		if slices.Contains(tierless, offer.Title) {
			t.Errorf("the answer holds %q", offer.Title)
		}
	}
}

func equalOffer(got, want offerJSON) bool {
	return got.ID == want.ID && got.Title == want.Title && got.Partner == want.Partner &&
		slices.Equal(got.Tiers, want.Tiers) && got.StartDate == want.StartDate &&
		got.EndDate == want.EndDate && got.URL == want.URL
}

// TBCOFFER-SC-007: offers_get_offer gives the terms of Oktopus, one line for each
// paragraph and no blank line.
func TestGetOfferOfTBC(t *testing.T) {
	t.Parallel()

	session := serve(t, tbcConfig(t))

	var detail detailJSON

	decode(
		t,
		call(
			t,
			session,
			getOffer,
			map[string]any{argBank: tbcCode, "id": oktopusID, argLanguage: "en"},
		),
		&detail,
	)

	lines := strings.Split(detail.Terms, "\n")
	if !slices.Contains(lines, "- The offer applies only to 6-month memberships.") ||
		slices.Contains(lines, "") {
		t.Errorf("terms = %q", detail.Terms)
	}

	if detail.Offer.URL != "https://tbcconcept.ge/en/offers/oktopusi-qeshbeqi" {
		t.Errorf("url = %q", detail.Offer.URL)
	}
}

// TBCOFFER-SC-010: With TBC off and BOG on, a search of TBC names the banks it accepts.
func TestSearchOfTBCWhenOff(t *testing.T) {
	t.Parallel()

	cfg := config(newBOGServer(t))
	cfg.TBC = offers.TBCConfig{Enabled: false}

	result := call(t, serve(t, cfg), searchOffers, map[string]any{argBanks: []string{tbcCode}})

	want := `banks: "tbc" is not a bank of this server. Use one of: bog.`

	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("result = %+v, want a tool error", result)
	}

	if text := textOf(result); text != want {
		t.Errorf("error = %q, want %q", text, want)
	}
}
