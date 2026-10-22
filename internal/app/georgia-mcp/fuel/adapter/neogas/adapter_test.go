package neogas_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/neogas"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	neogasclient "github.com/abgeo/georgia-mcp/pkg/neogas"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const testdata = "../../../../../../pkg/neogas/testdata"

func readAt() time.Time {
	return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
}

func adapterFor(t *testing.T, body string, status int) *neogas.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return neogas.NewAdapter(neogasclient.New(vendorhttp.WithBaseURL(server.URL)), readAt)
}

func recorded(t *testing.T, name string) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

func page(prices ...string) string {
	var body strings.Builder

	body.WriteString("<html><body>")

	for _, price := range prices {
		body.WriteString(`<span class="gas_price">` + price + `</span>`)
	}

	body.WriteString("</body></html>")

	return body.String()
}

// NEOFUEL-SC-004: The page gives one price of CNG with its names, no self-service price,
// and no date.
func TestPricesMapsTheRecordedPage(t *testing.T) {
	t.Parallel()

	adapter := adapterFor(t, recorded(t, "home.html"), http.StatusOK)

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	type digest struct {
		Code, Georgian, English, Standard string
		Grade                             domain.Grade
		SelfService, Dated                bool
	}

	got := make([]digest, 0, len(list.Prices))
	for _, price := range list.Prices {
		got = append(got, digest{
			Code:        price.Code,
			Georgian:    price.Name.Georgian,
			English:     price.Name.English,
			Standard:    price.Standard.StringFixed(-price.Standard.Exponent()),
			Grade:       price.Grade,
			SelfService: price.SelfService.Valid,
			Dated:       !price.EffectiveDate.IsZero(),
		})
	}

	want := []digest{{
		Code: "cng", Georgian: "ბუნებრივი აირი", English: "CNG", Standard: "1.60",
		Grade: domain.GradeCNG,
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}

	wantCompany := domain.Company{Code: "neogas", Name: "Neogas", PublishesEffectiveDate: false}
	if got := adapter.Company(); got != wantCompany {
		t.Errorf("Company() = %+v, want %+v", got, wantCompany)
	}
}

// NEOFUEL-SC-005: Every failure of the vendor or of its page wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "no gas price", status: http.StatusOK, body: page()},
		{
			name:   "two values that differ",
			status: http.StatusOK,
			body:   recorded(t, "home-split.html"),
		},
		{name: "not a number", status: http.StatusOK, body: page("1,60")},
		{name: "zero", status: http.StatusOK, body: page("0.00")},
		{name: "503", status: http.StatusServiceUnavailable, body: "busy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := adapterFor(t, tt.body, tt.status).Prices(t.Context())
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("Prices() error = %v, want kernel.ErrSourceUnavailable", err)
			}
		})
	}
}

// countingTransport serves the recorded page with no network, and counts the calls.
type countingTransport struct {
	body  []byte
	calls atomic.Int32
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       io.NopCloser(bytes.NewReader(c.body)),
		Request:    req,
	}, nil
}

// NEOFUEL-SC-007: The cache sends one call to Neogas in 30 minutes, for any count of
// requests, and a second call after 30 minutes.
func TestTheCacheBoundsTheCalls(t *testing.T) {
	t.Parallel()

	body := []byte(recorded(t, "home.html"))

	synctest.Test(t, func(t *testing.T) {
		transport := &countingTransport{body: body}
		client := neogasclient.New(
			vendorhttp.WithBaseURL("https://neogas.test"),
			vendorhttp.WithHTTPClient(&http.Client{Transport: transport}),
		)

		prices, err := cache.New(neogas.NewAdapter(client, time.Now))
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _ = prices.Close() })

		for range 10 {
			if _, err := prices.Prices(t.Context()); err != nil {
				t.Fatalf("Prices() error = %v", err)
			}

			time.Sleep(29 * time.Minute / 10)
		}

		if calls := transport.calls.Load(); calls != 1 {
			t.Errorf("calls in 29 minutes = %d, want 1", calls)
		}

		time.Sleep(2 * time.Minute)

		if _, err := prices.Prices(t.Context()); err != nil {
			t.Fatalf("Prices() at 31 minutes error = %v", err)
		}

		if calls := transport.calls.Load(); calls != 2 {
			t.Errorf("calls at 31 minutes = %d, want 2", calls)
		}
	})
}
