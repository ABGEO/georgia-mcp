package lukoil_test

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
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/lukoil"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	lukoilclient "github.com/abgeo/georgia-mcp/pkg/lukoil"
)

const instant = "2026-10-05 10:01:46"

const testdata = "../../../../../../pkg/lukoil/testdata"

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

func adapterFor(t *testing.T, body string, status int, sink *syncBuffer) *lukoil.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return lukoil.NewAdapter(
		lukoilclient.New(lukoilclient.WithBaseURL(server.URL)),
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

// page returns a price history page with the head of Lukoil and one row.
func page(row ...string) string {
	var cells strings.Builder

	for _, cell := range row {
		cells.WriteString("<td>" + cell + "</td>")
	}

	return `<table><thead><tr><th>#</th><th>Super Ecto 100</th><th>Euro Diesel</th>` +
		`<th>შეცვლის თარიღი</th></tr></thead><tbody><tr>` + cells.String() +
		`</tr></tbody></table>`
}

func exact(value decimal.Decimal) string {
	return value.StringFixed(-value.Exponent())
}

type digest struct {
	Grade       domain.Grade
	Standard    string
	SelfService string
	English     string
	Date        string
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
			English:     price.Name.English,
			Date:        price.EffectiveDate.String(),
		}
	}

	return digests
}

// LUKFUEL-SC-004: The adapter takes the newest row, leaves out the product at zero, and
// gives each price its grade, its name, and the date of the change.
func TestPricesMapsTheRecordedPage(t *testing.T) {
	t.Parallel()

	adapter := adapterFor(t, recorded(t, "prices-history.html"), http.StatusOK, &syncBuffer{})

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	changed := "2026-10-05"

	want := map[string]digest{
		"super_ecto": {
			Grade: domain.GradeSuper, Standard: "4.60", English: "Super Ecto", Date: changed,
		},
		"premium_avangard": {
			Grade: domain.GradePremium, Standard: "4.25", English: "Premium Avangard",
			Date: changed,
		},
		"euro_regular": {
			Grade: domain.GradeRegular, Standard: "3.95", English: "Euro Regular", Date: changed,
		},
		"euro_diesel": {
			Grade: domain.GradeDiesel, Standard: "4.97", English: "Euro Diesel", Date: changed,
		},
	}
	if diff := cmp.Diff(want, digestsOf(list)); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}

	wantCompany := domain.Company{Code: "lukoil", Name: "Lukoil", PublishesEffectiveDate: true}
	if got := adapter.Company(); got != wantCompany {
		t.Errorf("Company() = %+v, want %+v", got, wantCompany)
	}
}

// LUKFUEL-SC-005: Of two rows of one instant, the adapter takes the later in the page,
// and an unknown product gets the grade other with one warning.
func TestPricesMapsTheEdgePage(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(t, recorded(t, "prices-history-edge.html"), http.StatusOK, sink)

	if _, err := adapter.Prices(t.Context()); err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	got := digestsOf(list)

	if got["euro_diesel"].Standard != "4.99" {
		t.Errorf("euro_diesel = %s, want 4.99 from the later row", got["euro_diesel"].Standard)
	}

	if got["hydro"].Grade != domain.GradeOther {
		t.Errorf("hydro grade = %q, want other", got["hydro"].Grade)
	}

	if warnings := strings.Count(sink.String(), `"name":"Hydro"`); warnings != 1 {
		t.Errorf("the log holds %d warnings about Hydro, want 1:\n%s", warnings, sink)
	}
}

// LUKFUEL-SC-006: Every failure of the vendor or of its newest row wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "every price at zero",
			status: http.StatusOK,
			body:   page("1", "0.00", "0.00", instant),
		},
		{name: "negative", status: http.StatusOK, body: page("1", "0.00", "-1.00", instant)},
		{name: "not a number", status: http.StatusOK, body: page("1", "0.00", "4,97", instant)},
		{
			name:   "an instant that does not parse",
			status: http.StatusOK,
			body:   page("1", "0.00", "4.97", "05.10.2026"),
		},
		{name: "no row", status: http.StatusOK, body: page()},
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
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       io.NopCloser(bytes.NewReader(c.body)),
		Request:    req,
	}, nil
}

// LUKFUEL-SC-008: The cache sends one call to Lukoil in 30 minutes, for any count of
// requests, and a second call after 30 minutes.
func TestTheCacheBoundsTheCalls(t *testing.T) {
	t.Parallel()

	body := []byte(recorded(t, "prices-history.html"))

	synctest.Test(t, func(t *testing.T) {
		transport := &countingTransport{body: body}
		client := lukoilclient.New(
			lukoilclient.WithBaseURL("https://lukoil.test"),
			lukoilclient.WithHTTPClient(&http.Client{Transport: transport}),
		)

		prices, err := cache.New(
			lukoil.NewAdapter(client, time.Now, slog.New(slog.DiscardHandler)),
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
