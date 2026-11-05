package httpclient_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abgeo/georgia-mcp/internal/pkg/httpclient"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

func countingServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var count atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	return server, &count
}

func get(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("building the request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending the request: %w", err)
	}

	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("closing the body: %w", err)
	}

	return nil
}

// PLATFORM-SC-025: A full vendor limit waits for a slot, and fails as an unavailable
// source when the deadline comes first.
func TestVendorLimit(t *testing.T) {
	t.Parallel()

	cfg := httpclient.Config{Timeout: 10 * time.Second, Rate: 1, Burst: 1}

	t.Run("the second request waits", func(t *testing.T) {
		t.Parallel()

		server, count := countingServer(t)
		client := httpclient.New(cfg, slog.New(slog.DiscardHandler))

		start := time.Now()

		for range 2 {
			if err := get(t.Context(), client, server.URL); err != nil {
				t.Fatalf("get() error = %v", err)
			}
		}

		if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
			t.Errorf("two requests took %v, want about 1s", elapsed)
		}

		if got := count.Load(); got != 2 {
			t.Errorf("the server counted %d requests, want 2", got)
		}
	})

	t.Run("the deadline comes first", func(t *testing.T) {
		t.Parallel()

		server, count := countingServer(t)
		client := httpclient.New(cfg, slog.New(slog.DiscardHandler))

		if err := get(t.Context(), client, server.URL); err != nil {
			t.Fatalf("first get() error = %v", err)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		err := get(ctx, client, server.URL)
		if !errors.Is(err, kernel.ErrSourceUnavailable) {
			t.Errorf("second get() error = %v, want ErrSourceUnavailable", err)
		}

		if got := count.Load(); got != 1 {
			t.Errorf("the server counted %d requests, want 1", got)
		}
	})
}

// countingTransport answers 200 with no network, and counts the calls.
type countingTransport struct {
	calls atomic.Int32
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)

	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
}

// GULFUEL-SC-012: The vendor limit applies over a transport that the caller gives.
func TestVendorLimitOverATransport(t *testing.T) {
	t.Parallel()

	transport := &countingTransport{}
	client := httpclient.NewWithTransport(
		httpclient.Config{Timeout: 10 * time.Second, Rate: 1, Burst: 1},
		transport,
		slog.New(slog.DiscardHandler),
	)

	if err := get(t.Context(), client, "https://vendor.test"); err != nil {
		t.Fatalf("the first request: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	err := get(ctx, client, "https://vendor.test")
	if !errors.Is(err, kernel.ErrSourceUnavailable) {
		t.Errorf("the second request error = %v, want kernel.ErrSourceUnavailable", err)
	}

	if calls := transport.calls.Load(); calls != 1 {
		t.Errorf("the transport counts %d calls, want 1", calls)
	}
}

const (
	levelDebug = "DEBUG"
	levelWarn  = "WARN"
)

// vendorCall is the outcome that a test expects of one call to the vendor.
type vendorCall struct {
	level     string
	status    int
	errorType string
}

// logVendorCall sends one call and returns its record. A handler gets the cancel of the
// call, so it can cancel the call in flight.
func logVendorCall(
	t *testing.T,
	level slog.Level,
	serve func(w http.ResponseWriter, r *http.Request, cancel context.CancelFunc),
	closed bool,
) (map[string]any, string) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serve(w, r, cancel)
	}))
	t.Cleanup(server.Close)

	if closed {
		server.Close()
	}

	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level})).
		With(slog.String("vendor", "Acme"))
	client := httpclient.New(httpclient.Defaults().WithLimit(100, 10), logger)

	address := server.URL + "/rates?q=secret-value"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}

	if resp, err := client.Do(req); err == nil {
		_ = resp.Body.Close()
	}

	text := buf.String()

	var record map[string]any
	if err := json.Unmarshal([]byte(text), &record); err != nil {
		t.Fatalf("the log is not one JSON record: %q", text)
	}

	return record, text
}

func assertVendorRecord(t *testing.T, record map[string]any, text string, want vendorCall) {
	t.Helper()

	for key, value := range map[string]any{
		"msg":                 "vendor request",
		"vendor":              "Acme",
		"http.request.method": http.MethodGet,
		"server.address":      "127.0.0.1",
		"level":               want.level,
	} {
		if record[key] != value {
			t.Errorf("%s = %v, want %v: %s", key, record[key], value, text)
		}
	}

	if _, ok := record["event.duration"]; !ok {
		t.Errorf("the record has no event.duration: %s", text)
	}

	status, _ := record["http.response.status_code"].(float64)
	if int(status) != want.status {
		t.Errorf("http.response.status_code = %v, want %d", status, want.status)
	}

	errorType, _ := record["error.type"].(string)

	switch {
	case want.status == 0 && errorType == "":
		t.Error("a call with no answer has no error.type")
	case want.status != 0 && errorType != want.errorType:
		t.Errorf("error.type = %q, want %q", errorType, want.errorType)
	}

	if strings.Contains(text, "secret-value") {
		t.Errorf("the record holds the query: %s", text)
	}
}

// PLATFORM-SC-040: Each call to a vendor writes one record. Its level follows the outcome.
// No record holds the query, and only a debug record holds the path.
func TestVendorCallRecord(t *testing.T) {
	t.Parallel()

	type handler = func(http.ResponseWriter, *http.Request, context.CancelFunc)

	status := func(code int) handler {
		return func(w http.ResponseWriter, _ *http.Request, _ context.CancelFunc) {
			w.WriteHeader(code)
		}
	}

	cancelInFlight := func(_ http.ResponseWriter, r *http.Request, cancel context.CancelFunc) {
		cancel()
		<-r.Context().Done()
	}

	tests := []struct {
		name   string
		serve  handler
		closed bool
		want   vendorCall
	}{
		{"200", status(http.StatusOK), false, vendorCall{levelDebug, 200, ""}},
		{"404", status(http.StatusNotFound), false, vendorCall{"INFO", 404, "404"}},
		{"503", status(http.StatusServiceUnavailable), false, vendorCall{levelWarn, 503, "503"}},
		{"401", status(http.StatusUnauthorized), false, vendorCall{levelWarn, 401, "401"}},
		{"closed server", status(http.StatusOK), true, vendorCall{level: levelWarn}},
		{"cancelled", cancelInFlight, false, vendorCall{level: levelDebug}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record, text := logVendorCall(t, slog.LevelDebug, tt.serve, tt.closed)
			assertVendorRecord(t, record, text, tt.want)

			if record["url.path"] != "/rates" {
				t.Errorf("url.path = %v, want /rates", record["url.path"])
			}
		})
	}

	t.Run("info level", func(t *testing.T) {
		t.Parallel()

		serve := status(http.StatusServiceUnavailable)
		record, text := logVendorCall(t, slog.LevelInfo, serve, false)
		assertVendorRecord(t, record, text, vendorCall{levelWarn, 503, "503"})

		if _, ok := record["url.path"]; ok {
			t.Errorf("an info logger wrote url.path: %s", text)
		}
	})
}
