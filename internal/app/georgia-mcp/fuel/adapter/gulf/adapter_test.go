package gulf_test

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
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/gulf"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gulfclient "github.com/abgeo/georgia-mcp/pkg/gulf"
)

const testdata = "../../../../../../pkg/gulf/testdata"

// changed is the date of the newest change of each recorded page.
const changed = "2026-10-19"

func readAt() time.Time {
	return time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
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

func adapterFor(t *testing.T, body string, status int, sink *syncBuffer) *gulf.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return gulf.NewAdapter(
		gulfclient.New(
			gulfclient.WithBaseURL(server.URL),
			gulfclient.WithHTTPClient(server.Client()),
		),
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

// headCell returns a cell of the head: the label in a span of the class normal, for the
// date and the tier Gulf, as the page writes it.
func headCell(class, name string) string {
	return `<td class="` + class + `"><a><span class="normal">` + name +
		`</span><span class="small">X</span></a></td>`
}

// page returns a fuel prices page with a date column, the columns of the classes, and
// one row.
func page(classes []string, row ...string) string {
	var head, cells strings.Builder

	head.WriteString(headCell("first", "Date"))

	for _, class := range classes {
		head.WriteString(headCell(class, "Euro Diesel"))
	}

	for _, cell := range row {
		cells.WriteString("<td><span>" + cell + "</span></td>")
	}

	body := ""
	if len(row) > 0 {
		body = `<tr class="prices_cnt odd">` + cells.String() + `</tr>`
	}

	return `<table id="price_archive_tbl"><thead><tr class="prices_header">` + head.String() +
		`</tr></thead><tbody>` + body + `</tbody></table>`
}

func exact(value decimal.Decimal) string {
	return value.StringFixed(-value.Exponent())
}

type digest struct {
	Grade       domain.Grade
	Standard    string
	SelfService string
	English     string
	Georgian    string
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
			Georgian:    price.Name.In(kernel.LanguageGeorgian),
			Date:        price.EffectiveDate.String(),
		}
	}

	return digests
}

// GULFUEL-SC-005: The adapter takes the newest row and the tier Gulf only, and gives each
// price its grade, its names, the date of the change, and no self-service price.
func TestPricesMapsTheRecordedPage(t *testing.T) {
	t.Parallel()

	adapter := adapterFor(t, recorded(t, "fuel-prices.html"), http.StatusOK, &syncBuffer{})

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	want := map[string]digest{
		"g_force_super": {
			Grade: domain.GradeSuper, Standard: "4.69", English: "G-Force Super",
			Georgian: "G-Force სუპერი", Date: changed,
		},
		"g_force_premium": {
			Grade: domain.GradePremium, Standard: "4.39", English: "G-Force Premium",
			Georgian: "G-Force პრემიუმი", Date: changed,
		},
		"g_force_euro_regular": {
			Grade: domain.GradeRegular, Standard: "4.09", English: "G-Force Euro Regular",
			Georgian: "G-Force ევრო რეგულარი", Date: changed,
		},
		"g_force_euro_diesel": {
			Grade: domain.GradeDiesel, Standard: "5.05", English: "G-Force Euro Diesel",
			Georgian: "G-Force ევრო დიზელი", Date: changed,
		},
		"euro_diesel": {
			Grade: domain.GradeDiesel, Standard: "4.99", English: "Euro Diesel",
			Georgian: "ევრო დიზელი", Date: changed,
		},
	}
	if diff := cmp.Diff(want, digestsOf(list)); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}

	wantCompany := domain.Company{Code: "gulf", Name: "Gulf", PublishesEffectiveDate: true}
	if got := adapter.Company(); got != wantCompany {
		t.Errorf("Company() = %+v, want %+v", got, wantCompany)
	}
}

// GULFUEL-SC-006: The adapter takes the latest date with no trust in the order of the
// page, and the earlier of two rows of that date. It leaves out the product at zero, and
// gives an unknown product the grade other with one warning.
func TestPricesMapsTheEdgePage(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(t, recorded(t, "fuel-prices-edge.html"), http.StatusOK, sink)

	if _, err := adapter.Prices(t.Context()); err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	got := digestsOf(list)

	if date := got["euro_diesel"].Date; date != changed {
		t.Errorf("the effective date = %s, want %s, the latest date", date, changed)
	}

	if standard := got["euro_diesel"].Standard; standard != "4.99" {
		t.Errorf("euro_diesel = %s, want 4.99 from the earlier row of the date", standard)
	}

	if _, ok := got["g_force_super"]; ok {
		t.Errorf("the answer holds g_force_super at %s, want none", got["g_force_super"].Standard)
	}

	wantHydro := digest{
		Grade: domain.GradeOther, Standard: "3.20", English: "G-Force Hydro",
		Georgian: "G-Force Hydro", Date: changed,
	}
	if diff := cmp.Diff(wantHydro, got["g_force_hydro"]); diff != "" {
		t.Errorf("g_force_hydro (-want +got):\n%s", diff)
	}

	if warnings := strings.Count(sink.String(), `"name":"G-Force Hydro"`); warnings != 1 {
		t.Errorf("the log holds %d warnings about G-Force Hydro, want 1:\n%s", warnings, sink)
	}
}

// GULFUEL-SC-007: Every failure of the vendor, of its head, or of its newest row wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	tiers := []string{"g_col", "gp_col"}

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "every price of the tier at zero",
			status: http.StatusOK,
			body:   page(tiers, changed, "0.00", "4.89"),
		},
		{name: "negative", status: http.StatusOK, body: page(tiers, changed, "-1.00", "4.89")},
		{
			name:   "not a number",
			status: http.StatusOK,
			body:   page(tiers, changed, "4,99", "4.89"),
		},
		{
			name:   "a date that does not parse",
			status: http.StatusOK,
			body:   page(tiers, "19.10.2026", "4.99", "4.89"),
		},
		{
			name:   "no column of the tier",
			status: http.StatusOK,
			body:   page([]string{"gp_col"}, changed, "4.89"),
		},
		{
			name:   "no date column",
			status: http.StatusOK,
			body: `<table id="price_archive_tbl"><thead><tr>` + headCell("g_col", "Euro Diesel") +
				`</tr></thead><tbody><tr><td>4.99</td></tr></tbody></table>`,
		},
		{name: "no row", status: http.StatusOK, body: page(tiers)},
		{name: "403", status: http.StatusForbidden, body: "refused"},
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

// GULFUEL-SC-009: The cache sends one call to Gulf in 30 minutes, for any count of
// requests, and a second call after 30 minutes.
func TestTheCacheBoundsTheCalls(t *testing.T) {
	t.Parallel()

	body := []byte(recorded(t, "fuel-prices.html"))

	synctest.Test(t, func(t *testing.T) {
		transport := &countingTransport{body: body}
		client := gulfclient.New(
			gulfclient.WithBaseURL("https://gulf.test"),
			gulfclient.WithHTTPClient(&http.Client{Transport: transport}),
		)

		prices, err := cache.New(
			gulf.NewAdapter(client, time.Now, slog.New(slog.DiscardHandler)),
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
