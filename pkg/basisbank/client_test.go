package basisbank_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/basisbank"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	recordedAnswer = "testdata/website-xrates.json"
	dollar         = "USD"
)

func recorded(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile(recordedAnswer)
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func serve(t *testing.T, status int, body string) *basisbank.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return basisbank.New(vendorhttp.WithBaseURL(server.URL))
}

// BASRATE-SC-001: The client sends a POST with an empty body, the User-Agent of the
// repository, no token, and no key.
func TestRatesSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		_, _ = w.Write([]byte(recorded(t)))
	}))
	t.Cleanup(server.Close)

	if _, err := basisbank.New(vendorhttp.WithBaseURL(server.URL)).Rates(t.Context()); err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if got.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", got.Method)
	}

	if got.URL.Path != "/bb/websiteXrates" {
		t.Errorf("path = %q, want /bb/websiteXrates", got.URL.Path)
	}

	if got.ContentLength != 0 {
		t.Errorf("Content-Length = %d, want 0", got.ContentLength)
	}

	if accept := got.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}

	for _, name := range []string{"X-Csrf-Token", "Apikey", "Authorization", "X-Api-Key"} {
		if value := got.Header.Get(name); value != "" {
			t.Errorf("%s = %q, want no such header", name, value)
		}
	}
}

// BASRATE-SC-002: The client reads 2 days, and decodes the text of the rates of each.
func TestRatesReadsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	days, err := serve(t, http.StatusOK, recorded(t)).Rates(t.Context())
	if err != nil {
		t.Fatalf("Rates() error = %v", err)
	}

	if len(days) != 2 {
		t.Fatalf("Rates() = %d days, want 2", len(days))
	}

	first := days[0]
	if first.Date != "2026-10-25" {
		t.Errorf("Date = %q, want 2026-10-25", first.Date)
	}

	for name, pair := range map[string][2]string{
		"KursBuy[USD]":  {first.XRates.KursBuy[dollar], "2.5600"},
		"KursSell[USD]": {first.XRates.KursSell[dollar], "2.6300"},
		"Kurs[CNY]":     {first.XRates.Kurs["CNY"], "3.8939"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %q", name, pair[0], pair[1])
		}
	}
}

// BASRATE-SC-003: A text of the rates that is not JSON is ErrMalformedRates. A 429 is an
// APIError. A body that is not JSON is an error that is not an APIError.
func TestRatesReportsABrokenAnswer(t *testing.T) {
	t.Parallel()

	t.Run("rates not json", func(t *testing.T) {
		t.Parallel()

		_, err := serve(t, http.StatusOK, `[{"date":"2026-10-25","xrates":"not json"}]`).
			Rates(t.Context())
		if !errors.Is(err, basisbank.ErrMalformedRates) {
			t.Errorf("Rates() error = %v, want ErrMalformedRates", err)
		}
	})

	t.Run("status 429", func(t *testing.T) {
		t.Parallel()

		_, err := serve(t, http.StatusTooManyRequests, strings.Repeat("x", 2048)).
			Rates(t.Context())

		apiErr, ok := errors.AsType[*basisbank.APIError](err)
		if !ok {
			t.Fatalf("Rates() error = %v, want an APIError", err)
		}

		if apiErr.StatusCode != http.StatusTooManyRequests ||
			apiErr.Body != strings.Repeat("x", 1024) {
			t.Errorf(
				"APIError = %d with %d bytes, want 429 with 1024",
				apiErr.StatusCode,
				len(apiErr.Body),
			)
		}

		if !errors.Is(err, basisbank.ErrUnexpectedStatus) {
			t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
		}
	})

	t.Run("body not json", func(t *testing.T) {
		t.Parallel()

		_, err := serve(t, http.StatusOK, "[").Rates(t.Context())
		if _, ok := errors.AsType[*basisbank.APIError](err); err == nil || ok {
			t.Errorf("Rates() error = %v, want an error that is not an APIError", err)
		}
	})
}
