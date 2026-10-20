//go:build live

package offers_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers"
)

func mustURL(t *testing.T, text string) url.URL {
	t.Helper()

	base, err := url.Parse(text)
	if err != nil {
		t.Fatal(err)
	}

	return *base
}

// checkColdSearch searches one bank with an empty cache, within a limit of time.
func checkColdSearch(t *testing.T, cfg offers.Config, bank string, limit time.Duration) {
	t.Helper()

	session := serve(t, cfg)
	start := time.Now()

	var found searchJSON

	decode(t, call(t, session, searchOffers, map[string]any{argBanks: []string{bank}}), &found)

	if elapsed := time.Since(start); elapsed >= limit {
		t.Errorf("search of %s = %v, want less than %v", bank, elapsed, limit)
	}

	if len(found.Banks) != 1 || found.Banks[0].Total == 0 {
		t.Errorf("banks = %+v, want %s with offers", found.Banks, bank)
	}
}

// BOGOFFER-SC-021: With the real BOG and an empty cache, a search answers in less than
// 10 seconds.
func TestLiveSearchOfBOG(t *testing.T) {
	t.Parallel()

	checkColdSearch(t, offers.Config{BOG: offers.BOGConfig{
		Enabled: true,
		BaseURL: mustURL(t, "https://bankofgeorgia.ge/api/bog-b/offers-hub"),
		HTTP:    offers.BOGHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}}, bogCode, 10*time.Second)
}

// TBCOFFER-SC-009: With the real TBC and an empty cache, a search answers in less than
// 30 seconds.
func TestLiveSearchOfTBC(t *testing.T) {
	t.Parallel()

	checkColdSearch(t, offers.Config{TBC: offers.TBCConfig{
		Enabled: true,
		BaseURL: mustURL(t, "https://apigw.tbcbank.ge/api/v1/marketing"),
		HTTP:    offers.TBCHTTPConfig{Timeout: 10 * time.Second, Rate: 5, Burst: 5},
	}}, tbcCode, 30*time.Second)
}
