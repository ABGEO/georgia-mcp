package socar_test

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
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/socar"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	socarclient "github.com/abgeo/georgia-mcp/pkg/socar"
)

const testdata = "../../../../../../pkg/socar/testdata"

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

func adapterFor(t *testing.T, body string, status int, sink *syncBuffer) *socar.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return socar.NewAdapter(
		socarclient.New(socarclient.WithBaseURL(server.URL)),
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

// answer returns an answer of SOCAR with a status and the products.
func answer(status string, products ...string) string {
	return `{"GetCurrentPrices":{"Status":"` + status + `","Message":"Success","Results":[` +
		strings.Join(products, ",") + `]}}`
}

func product(code, price string) string {
	return `{"ActionDate":"2023-10-17T09:45:07","FuelNameGeo":"ნანო ევრო რეგულარი",` +
		`"FuelNameEng":"Nano Euro Regular","FuelUnitPrice":` + price + `,"FuelCode":"` + code + `"}`
}

func exact(value decimal.Decimal) string {
	return value.StringFixed(-value.Exponent())
}

type digest struct {
	Grade       domain.Grade
	Price       string
	English     string
	SelfService bool
	Dated       bool
}

func digestsOf(list app.PriceList) map[string]digest {
	digests := make(map[string]digest, len(list.Prices))

	for _, price := range list.Prices {
		digests[price.Code] = digest{
			Grade:       price.Grade,
			Price:       exact(price.Standard),
			English:     price.Name.English,
			SelfService: price.SelfService.Valid,
			Dated:       !price.EffectiveDate.IsZero(),
		}
	}

	return digests
}

// SOCFUEL-SC-005: The adapter gives each product its grade, keeps the digits, trims the
// names, and gives no self-service price and no date.
func TestPricesMapsTheRecordedAnswer(t *testing.T) {
	t.Parallel()

	list, err := adapterFor(t, recorded(t, "current-prices.json"), http.StatusOK, &syncBuffer{}).
		Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	want := map[string]digest{
		"SUPER":   {Grade: domain.GradeSuper, Price: "4.62", English: "Nano Super"},
		"PREMIUM": {Grade: domain.GradePremium, Price: "4.31", English: "Nano Premium"},
		"EURREG":  {Grade: domain.GradeRegular, Price: "4.02", English: "Nano Euro Regular"},
		"DIESEL":  {Grade: domain.GradeDiesel, Price: "4.9", English: "Euro 5 Diesel"},
		"EURODSL": {Grade: domain.GradeDiesel, Price: "4.96", English: "Nano Euro 5 Diesel"},
		"LPG":     {Grade: domain.GradeLPG, Price: "2.09", English: "LPG"},
		"CNG":     {Grade: domain.GradeCNG, Price: "1.6", English: "CNG"},
	}
	if diff := cmp.Diff(want, digestsOf(list)); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}
}

// SOCFUEL-SC-006: The adapter leaves out a product at zero, and gives an unknown code the
// grade other with one warning.
func TestPricesMapsTheEdgeAnswer(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(t, recorded(t, "current-prices-edge.json"), http.StatusOK, sink)

	if _, err := adapter.Prices(t.Context()); err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	got := digestsOf(list)

	if _, ok := got["A92"]; ok || len(got) != 7 {
		t.Errorf("prices = %v, want 7 and no A92", got)
	}

	if got["HYDRO"].Grade != domain.GradeOther {
		t.Errorf("HYDRO grade = %q, want other", got["HYDRO"].Grade)
	}

	if warnings := strings.Count(sink.String(), `"code":"HYDRO"`); warnings != 1 {
		t.Errorf("the log holds %d warnings about HYDRO, want 1:\n%s", warnings, sink)
	}
}

// SOCFUEL-SC-007: Every failure of the vendor or of its answer wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "status 0", status: http.StatusOK, body: answer("0", product("EURREG", "4.02"))},
		{name: "no status", status: http.StatusOK, body: `{}`},
		{name: "negative", status: http.StatusOK, body: answer("1", product("EURREG", "-1"))},
		{
			name:   "not a number",
			status: http.StatusOK,
			body:   answer("1", product("EURREG", `"abc"`)),
		},
		{
			name:   "every product at zero",
			status: http.StatusOK,
			body:   answer("1", product("EURREG", "0"), product("SUPER", "0")),
		},
		{name: "no product", status: http.StatusOK, body: answer("1")},
		{name: "400", status: http.StatusBadRequest, body: recorded(t, "error.json")},
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

// SOCFUEL-SC-008: SOCAR publishes no effective date.
func TestCompany(t *testing.T) {
	t.Parallel()

	got := adapterFor(t, "", http.StatusOK, &syncBuffer{}).Company()

	want := domain.Company{Code: "socar", Name: "SOCAR", PublishesEffectiveDate: false}
	if got != want {
		t.Errorf("Company() = %+v, want %+v", got, want)
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

// SOCFUEL-SC-013: The cache sends one call to SOCAR in 30 minutes, for any count of
// requests, and a second call after 30 minutes.
func TestTheCacheBoundsTheCalls(t *testing.T) {
	t.Parallel()

	body := []byte(recorded(t, "current-prices.json"))

	synctest.Test(t, func(t *testing.T) {
		transport := &countingTransport{body: body}
		client := socarclient.New(
			socarclient.WithBaseURL("https://socar.test"),
			socarclient.WithHTTPClient(&http.Client{Transport: transport}),
		)

		prices, err := cache.New(
			socar.NewAdapter(client, time.Now, slog.New(slog.DiscardHandler)),
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
