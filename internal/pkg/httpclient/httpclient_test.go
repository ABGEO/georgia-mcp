package httpclient_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		client := httpclient.New(cfg)

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
		client := httpclient.New(cfg)

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
