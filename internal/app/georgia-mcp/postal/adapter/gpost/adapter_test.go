package gpost_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/adapter/gpost"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/postal/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
	gpostclient "github.com/abgeo/georgia-mcp/pkg/gpost"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

const testdata = "../../../../../../pkg/gpost/testdata"

func readAt() time.Time {
	return time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC)
}

// vendor is a fake Georgian Post that serves the recorded page and one search answer,
// and records the cookie of each call.
type vendor struct {
	page   []byte
	result []byte
	status int

	mu      sync.Mutex
	cookies []string
}

func (v *vendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.mu.Lock()
	v.cookies = append(v.cookies, r.Header.Get("Cookie"))
	v.mu.Unlock()

	if v.status != 0 {
		w.WriteHeader(v.status)

		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Set-Cookie", "__RequestVerificationToken=test-token; path=/; HttpOnly")
		_, _ = w.Write(v.page)

		return
	}

	_, _ = w.Write(v.result)
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(testdata, name)) //nolint:gosec // A file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	return body
}

func adapterOf(t *testing.T, fake *vendor) *gpost.Adapter {
	t.Helper()

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	client := gpostclient.New(vendorhttp.WithBaseURL(server.URL))

	return gpost.NewAdapter(client, readAt)
}

func newVendor(t *testing.T, result string) *vendor {
	t.Helper()

	return &vendor{page: readFile(t, "postal-codes.html"), result: readFile(t, result)}
}

// POSTCODE-SC-022: The adapter sends the language, and converts each row to an address
// with its district and its postal code, read at the instant of now in Tbilisi.
func TestAddressesConvertsTheRows(t *testing.T) {
	t.Parallel()

	fake := newVendor(t, "find-rustavi-en.html")

	got, err := adapterOf(t, fake).Addresses(t.Context(), "Rustavi", kernel.LanguageEnglish)
	if err != nil {
		t.Fatalf("Addresses() error = %v", err)
	}

	if len(got.Addresses) != 12 {
		t.Fatalf("len(Addresses()) = %d, want 12", len(got.Addresses))
	}

	code, err := domain.ParsePostalCode("3700")
	if err != nil {
		t.Fatal(err)
	}

	want := domain.Address{Text: "City Rustavi", District: "Rustavi City", Code: code}
	if got.Addresses[0] != want {
		t.Errorf("first address = %+v, want %+v", got.Addresses[0], want)
	}

	_, offset := got.ReadAt.Zone()
	if _, tbilisi := readAt().In(kernel.TbilisiZone()).Zone(); !got.ReadAt.Equal(readAt()) ||
		offset != tbilisi {
		t.Errorf("ReadAt = %v, want %v in Tbilisi", got.ReadAt, readAt())
	}

	if got.Stale {
		t.Error("Stale = true, want false")
	}

	if fake.cookies[0] != "lang=en" {
		t.Errorf("page Cookie = %q, want lang=en", fake.cookies[0])
	}
}

// POSTCODE-SC-022: An answer with no match gives an empty list and no error.
func TestAddressesReadsAnEmptyAnswer(t *testing.T) {
	t.Parallel()

	got, err := adapterOf(t, newVendor(t, "find-none-ka.html")).
		Addresses(t.Context(), "ზზზზზქქ", kernel.LanguageGeorgian)
	if err != nil {
		t.Fatalf("Addresses() error = %v", err)
	}

	if len(got.Addresses) != 0 {
		t.Errorf("Addresses() = %+v, want none", got.Addresses)
	}
}

// POSTCODE-SC-010: A postal code that is not 4 digits, and an error of the client, wrap
// kernel.ErrSourceUnavailable.
func TestAddressesReportsAFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setUp func(*vendor)
	}{
		{
			name:  "short code",
			setUp: func(v *vendor) { v.result = readFile(t, "find-short-code.html") },
		},
		{
			name:  "malformed answer",
			setUp: func(v *vendor) { v.result = readFile(t, "find-malformed.html") },
		},
		{name: "status", setUp: func(v *vendor) { v.status = http.StatusInternalServerError }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := newVendor(t, "find-rustavi-en.html")
			tt.setUp(fake)

			_, err := adapterOf(t, fake).Addresses(t.Context(), "Rustavi", kernel.LanguageEnglish)
			if !errors.Is(err, kernel.ErrSourceUnavailable) {
				t.Errorf("Addresses() error = %v, want kernel.ErrSourceUnavailable", err)
			}
		})
	}
}
