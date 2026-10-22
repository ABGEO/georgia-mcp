package connect_test

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
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/connect"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	connectclient "github.com/abgeo/georgia-mcp/pkg/connect"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const testdata = "../../../../../../pkg/connect/testdata"

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

func adapterFor(t *testing.T, body string, status int, sink *syncBuffer) *connect.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return connect.NewAdapter(
		connectclient.New(vendorhttp.WithBaseURL(server.URL)),
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

// answer returns an answer of Connect with one product of the key diesel_10ppm.
func answer(standard, selfService string) string {
	return `{"date":"2026-10-21","prices":{"diesel_10ppm":{"label":"დიზელი 10PPM",` +
		`"standard":` + standard + `,"self_service":` + selfService + `}}}`
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

// CONFUEL-SC-004: The adapter gives each product its grade, its English name, the
// shortest decimal of each float, and no date.
func TestPricesMapsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	list, err := adapterFor(t, recorded(t, "current.json"), http.StatusOK, &syncBuffer{}).
		Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	want := map[string]digest{
		"super_ron98": {
			Grade: domain.GradeSuper, Standard: "4.35", SelfService: "4.25",
			Georgian: "სუპერი RON 98", English: "Super RON 98",
		},
		"ron96": {
			Grade: domain.GradePremium, Standard: "4.15", SelfService: "3.95",
			Georgian: "პრემიუმი RON 96", English: "Premium RON 96",
		},
		"regular_ron93": {
			Grade: domain.GradeRegular, Standard: "3.95", SelfService: "3.75",
			Georgian: "რეგულარი RON 93", English: "Regular RON 93",
		},
		"euro_diesel_10ppm": {
			Grade: domain.GradeDiesel, Standard: "4.9", SelfService: "4.75",
			Georgian: "ევრო დიზელი 10PPM", English: "Euro Diesel 10PPM",
		},
		"diesel_10ppm": {
			Grade: domain.GradeDiesel, Standard: "4.8", SelfService: "4.65",
			Georgian: "დიზელი 10PPM", English: "Diesel 10PPM",
		},
	}
	if diff := cmp.Diff(want, digestsOf(list)); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}

	wantCompany := domain.Company{Code: "connect", Name: "Connect", PublishesEffectiveDate: false}
	if got := adapterFor(t, "", http.StatusOK, &syncBuffer{}).Company(); got != wantCompany {
		t.Errorf("Company() = %+v, want %+v", got, wantCompany)
	}
}

// CONFUEL-SC-005: A null self-service price gives none, and an unknown key gets the grade
// other, its Georgian label in each language, and one warning.
func TestPricesMapsTheEdgeAnswer(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(t, recorded(t, "current-edge.json"), http.StatusOK, sink)

	if _, err := adapter.Prices(t.Context()); err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	got := digestsOf(list)

	if got["diesel_10ppm"].SelfService != "" {
		t.Errorf("diesel_10ppm self-service = %q, want none", got["diesel_10ppm"].SelfService)
	}

	hydro := list.Prices[codeIndex(list, "hydro")]
	if hydro.Grade != domain.GradeOther || hydro.Name.In(kernel.LanguageEnglish) != "წყალბადი" ||
		hydro.SelfService.Valid {
		t.Errorf("hydro = %+v, want other, its Georgian label in en, and no self-service", hydro)
	}

	if warnings := strings.Count(sink.String(), `"key":"hydro"`); warnings != 1 {
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

// CONFUEL-SC-006: Every failure of the vendor or of its answer wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "no product", status: http.StatusOK, body: `{"prices":{}}`},
		{name: "no list", status: http.StatusOK, body: `{}`},
		{name: "standard at zero", status: http.StatusOK, body: answer("0", "4.65")},
		{name: "not a number", status: http.StatusOK, body: answer(`"abc"`, "4.65")},
		{name: "negative self-service", status: http.StatusOK, body: answer("4.8", "-1")},
		{name: "404", status: http.StatusNotFound, body: `{"message":"No prices"}`},
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
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(c.body)),
		Request:    req,
	}, nil
}

// CONFUEL-SC-008: The cache sends one call to Connect in 30 minutes, for any count of
// requests, and a second call after 30 minutes.
func TestTheCacheBoundsTheCalls(t *testing.T) {
	t.Parallel()

	body := []byte(recorded(t, "current.json"))

	synctest.Test(t, func(t *testing.T) {
		transport := &countingTransport{body: body}
		client := connectclient.New(
			vendorhttp.WithBaseURL("https://connect.test"),
			vendorhttp.WithHTTPClient(&http.Client{Transport: transport}),
		)

		prices, err := cache.New(
			connect.NewAdapter(client, time.Now, slog.New(slog.DiscardHandler)),
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
