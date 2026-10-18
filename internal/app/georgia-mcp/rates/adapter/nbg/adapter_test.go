package nbg_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/adapter/nbg"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/rates/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	nbgclient "github.com/abgeo/georgia-mcp/pkg/nbg"
)

const testdata = "../../../../../../pkg/nbg/testdata"

func adapterFor(t *testing.T, handler http.HandlerFunc) *nbg.Adapter {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	readAt := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)

	return nbg.NewAdapter(
		nbgclient.New(nbgclient.WithBaseURL(server.URL)),
		func() time.Time { return readAt },
	)
}

func recorded(t *testing.T, name string) http.HandlerFunc {
	t.Helper()

	path := filepath.Join(testdata, name)

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }
}

func readSet(t *testing.T, name string) app.RateSetResult {
	t.Helper()

	date, _ := domain.ParseDate("2026-10-17")

	result, err := adapterFor(
		t,
		recorded(t, name),
	).RateSet(t.Context(), date, kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("RateSet() error = %v", err)
	}

	return result
}

// NBGRATE-SC-013: The change takes the digits of diffFormated and the sign of diff.
func TestRateSetMapsTheChange(t *testing.T) {
	t.Parallel()

	set := readSet(t, "en-2026-10-17.json").Set

	tests := map[domain.CurrencyCode]string{"EUR": "0.0145", "AED": "-0.0013"}

	for code, want := range tests {
		rate, ok := set.Find(code)
		if !ok {
			t.Fatalf("the set holds no %s", code)
		}

		if got := rate.Change.String(); got != want {
			t.Errorf("%s change = %s, want %s", code, got, want)
		}
	}
}

// NBGRATE-SC-015: The publication instant is the wall time of Tbilisi.
func TestRateSetReadsThePublicationInTbilisiTime(t *testing.T) {
	t.Parallel()

	result := readSet(t, "en-2026-10-17.json")

	want := time.Date(2026, 10, 16, 17, 1, 1, 982_000_000, domain.TbilisiZone())
	if !result.Set.PublishedAt.Equal(want) {
		t.Errorf("PublishedAt = %v, want %v", result.Set.PublishedAt, want)
	}

	if _, offset := result.Set.PublishedAt.Zone(); offset != 4*60*60 {
		t.Errorf("PublishedAt offset = %d, want UTC+4", offset)
	}
}

func TestRateSetMapsTheSet(t *testing.T) {
	t.Parallel()

	result := readSet(t, "en-2026-10-11.json")

	if result.Set.ValidFrom.String() != "2026-10-10" || len(result.Set.Rates) != 42 {
		t.Fatalf(
			"set = %s with %d rates, want 2026-10-10 with 42",
			result.Set.ValidFrom,
			len(result.Set.Rates),
		)
	}

	if result.ReadAt.IsZero() || result.Stale {
		t.Errorf(
			"ReadAt = %v, Stale = %v, want the read instant and false",
			result.ReadAt,
			result.Stale,
		)
	}

	jpy, _ := result.Set.Find("JPY")
	if jpy.Quantity != 100 || jpy.Rate.Exponent() != -4 {
		t.Errorf("JPY = %d units at %s, want 100 units with four decimals", jpy.Quantity, jpy.Rate)
	}
}

func TestRateSetMapsAnEmptyAnswer(t *testing.T) {
	t.Parallel()

	if result := readSet(t, "en-1990-01-01.json"); !result.Set.Empty() {
		t.Errorf("set = %v, want an empty set", result.Set)
	}
}

// NBGRATE-SC-021: A failure of the vendor wraps kernel.ErrSourceUnavailable.
func TestRateSetWrapsAFailure(t *testing.T) {
	t.Parallel()

	tests := map[string]http.HandlerFunc{
		"server error": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "down", http.StatusServiceUnavailable)
		},
		"body that is not JSON": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>"))
		},
		"rate that is not a number": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"date":"2026-10-17T00:00:00.000Z","currencies":[` +
				`{"code":"USD","quantity":1,"rateFormated":"n/a","diffFormated":"0","diff":0,` +
				`"date":"2026-10-16T17:01:01.982Z"}]}]`))
		},
	}

	date, _ := domain.ParseDate("2026-10-17")

	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := adapterFor(t, handler).RateSet(t.Context(), date, kernel.LanguageEnglish)
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("RateSet() error = %v, want ErrSourceUnavailable", err)
			}
		})
	}
}
