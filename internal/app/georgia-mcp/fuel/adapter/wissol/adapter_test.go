package wissol_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/adapter/wissol"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/fuel/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	wissolclient "github.com/abgeo/georgia-mcp/pkg/wissol"
)

const (
	testdata = "../../../../../../pkg/wissol/testdata"
	changed  = "2026-10-08"
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

func adapterFor(t *testing.T, body string, status int, sink *syncBuffer) *wissol.Adapter {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return wissol.NewAdapter(
		wissolclient.New(wissolclient.WithBaseURL(server.URL)),
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

func exact(value decimal.Decimal) string {
	return value.StringFixed(-value.Exponent())
}

type digest struct {
	Grade       domain.Grade
	Price       string
	SelfService string
	Effective   string
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
			Effective:   price.EffectiveDate.String(),
		}
	}

	return digests
}

// WISFUEL-SC-012: The adapter leaves out a product that is not active, gives an unknown
// code the grade other with one warning, reads a self-service price of zero as none,
// and keeps the digits and the date of each price.
func TestPricesMapsTheProducts(t *testing.T) {
	t.Parallel()

	sink := &syncBuffer{}
	adapter := adapterFor(t, recorded(t, "fuelprice-edge.json"), http.StatusOK, sink)

	for range 2 {
		if _, err := adapter.Prices(t.Context()); err != nil {
			t.Fatalf("Prices() error = %v", err)
		}
	}

	list, err := adapter.Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	want := map[string]digest{
		"02": {Grade: domain.GradePremium, Price: "4.32", Effective: changed},
		"03": {
			Grade:       domain.GradeDiesel,
			Price:       "5.07",
			SelfService: "4.97",
			Effective:   changed,
		},
		"05": {
			Grade:       domain.GradeRegular,
			Price:       "4.08",
			SelfService: "3.93",
			Effective:   changed,
		},
		"06": {
			Grade:       domain.GradeDiesel,
			Price:       "4.93",
			SelfService: "4.73",
			Effective:   changed,
		},
		"07": {Grade: domain.GradeSuper100, Price: "4.79", Effective: "2026-10-17"},
		"08": {
			Grade:       domain.GradeDiesel,
			Price:       "4.98",
			SelfService: "4.83",
			Effective:   changed,
		},
		"99": {Grade: domain.GradeOther, Price: "1.64", Effective: "2023-11-07"},
	}
	if diff := cmp.Diff(want, digestsOf(list)); diff != "" {
		t.Errorf("prices (-want +got):\n%s", diff)
	}

	if !list.ReadAt.Equal(readAt()) || list.Stale {
		t.Errorf("ReadAt = %v, Stale = %v, want %v and false", list.ReadAt, list.Stale, readAt())
	}

	if warnings := strings.Count(sink.String(), `"code":"99"`); warnings != 1 {
		t.Errorf("the log holds %d warnings about the code 99, want 1:\n%s", warnings, sink)
	}
}

// WISFUEL-SC-012: The English and the Georgian name of a product reach the domain.
func TestPricesKeepsBothNames(t *testing.T) {
	t.Parallel()

	list, err := adapterFor(t, recorded(t, "fuelprice.json"), http.StatusOK, &syncBuffer{}).
		Prices(t.Context())
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	for _, price := range list.Prices {
		if price.Code == "05" {
			want := domain.ProductName{Georgian: "ევრო რეგულარი", English: "Euro Regular"}
			if price.Name != want {
				t.Errorf("Name = %+v, want %+v", price.Name, want)
			}

			return
		}
	}

	t.Error("the list lacks the product 05")
}

// WISFUEL-SC-013: Every failure of the vendor or of its answer wraps
// kernel.ErrSourceUnavailable.
func TestPricesReportsAFailure(t *testing.T) {
	t.Parallel()

	product := func(fields string) string {
		return `[{"productCode":"05","fuelType":"ევრო რეგულარი","fuelTypeEng":"Euro Regular",` +
			`"isActive":true,` + fields + `}]`
	}

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "500", status: http.StatusInternalServerError, body: "error"},
		{name: "not json", status: http.StatusOK, body: "<html></html>"},
		{name: "no price", status: http.StatusOK, body: product(`"date":"2026-10-08T00:00:00"`)},
		{
			name:   "negative self-service",
			status: http.StatusOK,
			body:   product(`"price":4.08,"priceSelfService":-1,"date":"2026-10-08T00:00:00"`),
		},
		{
			name:   "date of another form",
			status: http.StatusOK,
			body:   product(`"price":4.08,"date":"08.10.2026"`),
		},
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
