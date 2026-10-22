package rompetrol_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/cache"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/rompetrol"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	rompetrolclient "github.com/abgeo/georgia-mcp/pkg/rompetrol"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const (
	testdata   = "../../../../../../pkg/rompetrol/testdata"
	superPrice = "4.65"
	superSelf  = "4.55"
)

func readAt() time.Time {
	return time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
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

func adapterFor(t *testing.T, body string, status int, sink *syncBuffer) *rompetrol.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return rompetrol.NewAdapter(
		rompetrolclient.New(vendorhttp.WithBaseURL(server.URL)),
		readAt,
		slog.New(slog.NewJSONHandler(sink, nil)),
	)
}

func recorded(t *testing.T, name string) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// page returns a short page with the anchor of the price table and the rows.
func page(rows ...[2]string) string {
	var html strings.Builder

	html.WriteString(`<html><body><a id="pricelist"></a><table><thead><tr><th>პროდუქტი</th>` +
		`<th>ლარი / ლ</th></tr></thead><tbody>`)

	for _, row := range rows {
		html.WriteString("<tr><td>" + row[0] + "</td><td>" + row[1] + "</td></tr>")
	}

	html.WriteString(`</tbody></table></body></html>`)

	return html.String()
}

func exact(value decimal.Decimal) string {
	return value.StringFixed(-value.Exponent())
}

type digest struct {
	Grade       domain.Grade
	Price       string
	SelfService string
	English     string
	Dated       bool
}

func digestsOf(list app.PriceList) map[string]digest {
	digests := make(map[string]digest, len(list.Prices))

	for _, price := range list.Prices {
		self := ""
		if price.SelfService.Valid {
			self = exact(price.SelfService.Decimal)
		}

		digests[price.Code] = digest{
			Grade:       price.Grade,
			Price:       exact(price.Standard),
			SelfService: self,
			English:     price.Name.English,
			Dated:       !price.EffectiveDate.IsZero(),
		}
	}

	return digests
}

// ROMFUEL-SC-006: The adapter pairs each product with the self-service row after it,
// gives the grade and the English name of each known product, and gives no date.
func TestPricesMapsTheRecordedPage(t *testing.T) {
	t.Parallel()

	list, err := adapterFor(t, recorded(t, "home.html"), http.StatusOK, &syncBuffer{}).
		Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	want := map[string]digest{
		"efix_super": {
			Grade:       domain.GradeSuper,
			Price:       superPrice,
			SelfService: superSelf,
			English:     "efix Super",
		},
		"efix_euro_premium": {
			Grade:       domain.GradePremium,
			Price:       "4.32",
			SelfService: "4.22",
			English:     "efix Euro Premium",
		},
		"efix_euro_regular": {
			Grade:       domain.GradeRegular,
			Price:       "4.02",
			SelfService: "3.92",
			English:     "efix Euro Regular",
		},
		"efix_euro_diesel": {
			Grade:       domain.GradeDiesel,
			Price:       "4.99",
			SelfService: "4.89",
			English:     "efix Euro Diesel",
		},
		"euro_diesel": {
			Grade: domain.GradeDiesel, Price: "4.92", SelfService: "4.82", English: "Euro Diesel",
		},
	}
	if diff := cmp.Diff(want, digestsOf(list)); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}

	for _, price := range list.Prices {
		if price.Code == "efix_euro_regular" && price.Name.Georgian != "efix ევრო რეგულარი" {
			t.Errorf("Georgian name = %q, want efix ევრო რეგულარი", price.Name.Georgian)
		}
	}
}

// ROMFUEL-SC-007: An unknown product gets the grade other with one warning, a product
// with no self-service row has no self-service price, and a self-service price of zero
// gives none.
func TestPricesMapsTheEdgePage(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(t, recorded(t, "home-edge.html"), http.StatusOK, sink)

	if _, err := adapter.Prices(t.Context()); err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	const unknown = "efix სუპერი პლიუსი"

	got := digestsOf(list)

	want := map[string]digest{
		unknown: {Grade: domain.GradeOther, Price: superPrice, SelfService: superSelf},
		"efix_euro_premium": {
			Grade:   domain.GradePremium,
			Price:   "4.32",
			English: "efix Euro Premium",
		},
		"euro_diesel": {Grade: domain.GradeDiesel, Price: "4.92", English: "Euro Diesel"},
	}
	for code, digest := range want {
		if diff := cmp.Diff(digest, got[code]); diff != "" {
			t.Errorf("%s (-want +got):\n%s", code, diff)
		}
	}

	if warnings := strings.Count(sink.String(), `"name":"`+unknown+`"`); warnings != 1 {
		t.Errorf("the log holds %d warnings about %s, want 1:\n%s", warnings, unknown, sink)
	}
}

// ROMFUEL-SC-008: Every failure of the vendor or of its page wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	const (
		super     = "efix სუპერი"
		selfSuper = "- efix სუპერი (თვითმომსახურების სადგურებზე)"
	)

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "self-service row first",
			status: http.StatusOK,
			body:   page([2]string{selfSuper, superSelf}),
		},
		{
			name:   "two self-service rows",
			status: http.StatusOK,
			body: page(
				[2]string{super, superPrice},
				[2]string{selfSuper, superSelf},
				[2]string{selfSuper, "4.50"},
			),
		},
		{name: "comma", status: http.StatusOK, body: page([2]string{super, "4,65"})},
		{name: "zero standard", status: http.StatusOK, body: page([2]string{super, "0"})},
		{
			name:   "negative self-service",
			status: http.StatusOK,
			body:   page([2]string{super, superPrice}, [2]string{selfSuper, "-1"}),
		},
		{name: "empty table", status: http.StatusOK, body: page()},
		{name: "no table", status: http.StatusOK, body: recorded(t, "home-no-table.html")},
		{name: "500", status: http.StatusInternalServerError, body: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := adapterFor(t, tt.body, tt.status, &syncBuffer{}).Prices(t.Context())
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("Prices() error = %v, want kernel.ErrSourceUnavailable", err)
			}
		})
	}
}

// ROMFUEL-SC-009: Rompetrol publishes no effective date.
func TestCompany(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, "", http.StatusOK, &syncBuffer{}).Company()

	want := domain.Company{Code: "rompetrol", Name: "Rompetrol", PublishesEffectiveDate: false}
	if got != want {
		t.Errorf("Company() = %+v, want %+v", got, want)
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
		Header:     http.Header{"Content-Type": []string{"text/html; charset=UTF-8"}},
		Body:       io.NopCloser(bytes.NewReader(c.body)),
		Request:    req,
	}, nil
}

// ROMFUEL-SC-014: The cache sends one call to Rompetrol in 30 minutes, for any count of
// requests, and a second call after 30 minutes.
func TestTheCacheBoundsTheCalls(t *testing.T) {
	t.Parallel()

	body := []byte(recorded(t, "home.html"))

	synctest.Test(t, func(t *testing.T) {
		transport := &countingTransport{body: body}
		client := rompetrolclient.New(
			vendorhttp.WithBaseURL("https://rompetrol.test"),
			vendorhttp.WithHTTPClient(&http.Client{Transport: transport}),
		)

		prices, err := cache.New(
			rompetrol.NewAdapter(client, time.Now, slog.New(slog.DiscardHandler)),
		)
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
