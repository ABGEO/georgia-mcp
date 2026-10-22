package offers_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers"
	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/vendors"
)

const (
	libertyTestdata = "../../../../pkg/liberty/testdata"
	libertyCode     = "liberty"
	libertyEmpty    = `{"data":{"list":[]},"succeeded":true,"message":""}`
)

// libertyServer serves the recorded answers of Liberty. A list with no file is empty.
// A key with a prefix in failing answers 500.
func libertyServer(t *testing.T, failing ...string) url.URL {
	t.Helper()

	names, err := filepath.Glob(filepath.Join(libertyTestdata, "*"))
	if err != nil {
		t.Fatal(err)
	}

	bodies := map[string][]byte{}

	for _, name := range names {
		body, err := os.ReadFile(name) //nolint:gosec // A file of testdata.
		if err != nil {
			t.Fatal(err)
		}

		base := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(name), ".json"), ".html")
		bodies[base] = body
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := libertyKey(r)

		if slices.ContainsFunc(
			failing,
			func(prefix string) bool { return strings.HasPrefix(key, prefix) },
		) {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		body, ok := bodies[key]
		if !ok && !strings.HasPrefix(key, "details-") {
			body, ok = []byte(libertyEmpty), true
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

func libertyKey(r *http.Request) string {
	switch path.Base(r.URL.Path) {
	case "loadcategories":
		return argCats
	case "loadtypes":
		return "types"
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

func libertyConfig(base url.URL) vendors.Config {
	return vendors.Config{
		Enabled: true,
		BaseURL: base,
		HTTP:    httpclient.Config{Timeout: 10 * time.Second, Rate: 100, Burst: 100},
	}
}

// LIBOFFER-SC-007: offers_get_offer gives the terms of offer 727, one line for each
// paragraph and no blank line.
func TestGetOfferOfLiberty(t *testing.T) {
	t.Parallel()

	session := serve(t, offers.Config{Liberty: libertyConfig(libertyServer(t))})

	var detail detailJSON

	decode(
		t,
		call(
			t,
			session,
			getOffer,
			map[string]any{argBank: libertyCode, "id": "727", argLanguage: "en"},
		),
		&detail,
	)

	lines := strings.Split(detail.Terms, "\n")
	if len(lines) != 13 || slices.Contains(lines, "") ||
		lines[12] != "• The offer is valid until December 31, 2026" {
		t.Errorf("terms = %q", detail.Terms)
	}

	if detail.Offer.URL != "https://studioliberty.ge/en/studio-lifestyle/offers/details/727" {
		t.Errorf("url = %q", detail.Offer.URL)
	}
}

// LIBOFFER-SC-009: When Liberty fails, a search of leisure answers for BOG and TBC, and
// names Liberty Bank as unavailable.
func TestSearchWhenLibertyFails(t *testing.T) {
	t.Parallel()

	cfg := config(newBOGServer(t))
	cfg.TBC = tbcConfig(t).TBC
	cfg.Liberty = libertyConfig(libertyServer(t, "type-2-"))

	var found struct {
		Banks []struct {
			Bank string `json:"bank"`
		} `json:"banks"`
		Unavailable []struct {
			Bank   string `json:"bank"`
			Source string `json:"source"`
		} `json:"unavailable"`
	}

	decode(
		t,
		call(t, serve(t, cfg), searchOffers, map[string]any{argCats: []string{"leisure"}}),
		&found,
	)

	banks := make([]string, 0, len(found.Banks))
	for _, bank := range found.Banks {
		banks = append(banks, bank.Bank)
	}

	if !slices.Equal(banks, []string{bogCode, tbcCode}) || len(found.Unavailable) != 1 ||
		found.Unavailable[0].Bank != libertyCode || found.Unavailable[0].Source != "Liberty Bank" {
		t.Errorf("banks = %v, unavailable = %+v", banks, found.Unavailable)
	}
}
