package portal_test

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
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/portal"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	portalclient "github.com/abgeo/georgia-mcp/pkg/portal"
)

const testdata = "../../../../../../pkg/portal/testdata"

func readAt() time.Time {
	return time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
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

func adapterFor(t *testing.T, body string, status int, sink *syncBuffer) *portal.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return portal.NewAdapter(
		portalclient.New(portalclient.WithBaseURL(server.URL)),
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

// answer returns a stream of Portal with one product of the slug euro-diesel.
func answer(price string) string {
	return `6:{"fuelPricesData":[{"name":"EURO DIESEL","latestPrice":"` + price +
		`","fuelType":{"slug":"euro-diesel"}}]}`
}

func exact(value decimal.Decimal) string {
	return value.StringFixed(-value.Exponent())
}

type digest struct {
	Grade       domain.Grade
	Standard    string
	SelfService string
	Georgian    string
	English     string
	Dated       bool
}

func digestsOf(list app.PriceList) map[string]digest {
	digests := make(map[string]digest, len(list.Prices))

	for _, price := range list.Prices {
		selfService := ""
		if price.SelfService.Valid {
			selfService = exact(price.SelfService.Decimal)
		}

		digests[price.Code] = digest{
			Grade:       price.Grade,
			Standard:    exact(price.Standard),
			SelfService: selfService,
			Georgian:    price.Name.Georgian,
			English:     price.Name.English,
			Dated:       !price.EffectiveDate.IsZero(),
		}
	}

	return digests
}

// PORFUEL-SC-004: The adapter gives each product its grade, its Georgian name, the
// standard price only, and no date.
func TestPricesMapsTheRecordedStream(t *testing.T) {
	t.Parallel()

	list, err := adapterFor(t, recorded(t, "fuel-prices.rsc"), http.StatusOK, &syncBuffer{}).
		Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	want := map[string]digest{
		"super": {
			Grade: domain.GradeSuper, Standard: "4.39", Georgian: "სუპერი", English: "SUPER",
		},
		"premium": {
			Grade: domain.GradePremium, Standard: "4.19", Georgian: "პრემიუმი", English: "PREMIUM",
		},
		"euro-regular": {
			Grade: domain.GradeRegular, Standard: "3.94",
			Georgian: "ევრო რეგულარი", English: "EURO REGULAR",
		},
		"euro-diesel": {
			Grade: domain.GradeDiesel, Standard: "4.94",
			Georgian: "ევრო დიზელი", English: "EURO DIESEL",
		},
		"effect-diesel": {
			Grade: domain.GradeDiesel, Standard: "4.84",
			Georgian: "ეფექტ დიზელი", English: "EFFECT DIESEL",
		},
	}
	if diff := cmp.Diff(want, digestsOf(list)); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}

	wantCompany := domain.Company{Code: "portal", Name: "Portal", PublishesEffectiveDate: false}
	if got := adapterFor(t, "", http.StatusOK, &syncBuffer{}).Company(); got != wantCompany {
		t.Errorf("Company() = %+v, want %+v", got, wantCompany)
	}
}

// PORFUEL-SC-005: An unknown slug gets the grade other, the name of Portal in each
// language, and one warning.
func TestPricesMapsTheEdgeStream(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(t, recorded(t, "fuel-prices-edge.rsc"), http.StatusOK, sink)

	if _, err := adapter.Prices(t.Context()); err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	hydro := list.Prices[codeIndex(list, "hydro")]
	if hydro.Grade != domain.GradeOther || hydro.Name.In(kernel.LanguageGeorgian) != "HYDRO" {
		t.Errorf("hydro = %+v, want other, and the name of Portal in ka", hydro)
	}

	if warnings := strings.Count(sink.String(), `"slug":"hydro"`); warnings != 1 {
		t.Errorf("the log holds %d warnings about hydro, want 1:\n%s", warnings, sink)
	}
}

func codeIndex(list app.PriceList, code string) int {
	for i, price := range list.Prices {
		if price.Code == code {
			return i
		}
	}

	return -1
}

// PORFUEL-SC-006: Every failure of the vendor or of its stream wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "no product", status: http.StatusOK, body: `6:{"fuelPricesData":[]}`},
		{name: "no list", status: http.StatusOK, body: `0:{"children":"x"}`},
		{name: "not a number", status: http.StatusOK, body: answer("4,39")},
		{name: "zero", status: http.StatusOK, body: answer("0")},
		{name: "500", status: http.StatusInternalServerError, body: "down"},
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

// countingTransport serves the recorded answer with no network, and counts the calls.
type countingTransport struct {
	body  []byte
	calls atomic.Int32
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/x-component"}},
		Body:       io.NopCloser(bytes.NewReader(c.body)),
		Request:    req,
	}, nil
}

// PORFUEL-SC-008: The cache sends one call to Portal in 30 minutes, for any count of
// requests, and a second call after 30 minutes.
func TestTheCacheBoundsTheCalls(t *testing.T) {
	t.Parallel()

	body := []byte(recorded(t, "fuel-prices.rsc"))

	synctest.Test(t, func(t *testing.T) {
		transport := &countingTransport{body: body}
		client := portalclient.New(
			portalclient.WithBaseURL("https://portal.test"),
			portalclient.WithHTTPClient(&http.Client{Transport: transport}),
		)

		prices, err := cache.New(
			portal.NewAdapter(client, time.Now, slog.New(slog.DiscardHandler)),
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
