//go:build live

package offers_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers"
)

// BOGOFFER-SC-021: With the real BOG and an empty cache, a search answers in less than
// 10 seconds.
func TestLiveSearchOfBOG(t *testing.T) {
	t.Parallel()

	base, err := url.Parse("https://bankofgeorgia.ge/api/bog-b/offers-hub")
	if err != nil {
		t.Fatal(err)
	}

	session := serve(t, offers.Config{BOG: offers.BOGConfig{
		Enabled: true,
		BaseURL: *base,
		HTTP:    offers.BOGHTTPConfig{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
	}})

	start := time.Now()

	var found searchJSON

	decode(t, call(t, session, searchOffers, map[string]any{"banks": []string{bogCode}}), &found)

	if elapsed := time.Since(start); elapsed >= 10*time.Second {
		t.Errorf("search = %v, want less than 10s", elapsed)
	}

	if len(found.Banks) != 1 || found.Banks[0].Total == 0 {
		t.Errorf("banks = %+v, want BOG with offers", found.Banks)
	}
}
