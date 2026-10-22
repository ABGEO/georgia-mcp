package connect_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/connect"
	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

func serveFile(t *testing.T, path string, status int) *httptest.Server {
	t.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // The path names a file of testdata.
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	return server
}

// CONFUEL-SC-001: The client sends the path, the Accept header, and the User-Agent.
func TestCurrentSendsTheRequest(t *testing.T) {
	t.Parallel()

	var got *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())

		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	if _, err := connect.New(vendorhttp.WithBaseURL(server.URL)).Current(t.Context()); err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if got.URL.Path != "/current" {
		t.Errorf("path = %q, want /current", got.URL.Path)
	}

	if accept := got.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := got.Header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp-connect/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp-connect/<version> (+<repository>)", agent)
	}
}

// CONFUEL-SC-001: The client decodes a recorded answer and keeps the digits of the answer.
func TestCurrentDecodesARecordedAnswer(t *testing.T) {
	t.Parallel()

	server := serveFile(t, "testdata/current.json", http.StatusOK)

	got, err := connect.New(vendorhttp.WithBaseURL(server.URL)).Current(t.Context())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if got.Date != "2026-10-21" || len(got.Prices) != 5 {
		t.Fatalf("Date = %q with %d products, want 2026-10-21 with 5", got.Date, len(got.Prices))
	}

	super := got.Prices["super_ron98"]
	if super.Label != "სუპერი RON 98" ||
		super.Standard.String() != "4.3499999999999996447286321199499070644378662109375" ||
		super.SelfService.String() != "4.25" {
		t.Errorf("super_ron98 = %+v, want the label and the digits of the answer", super)
	}
}

// CONFUEL-SC-002: A status other than 200 is an APIError with the first 1 KiB of the body.
func TestCurrentReportsAStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"No prices"}`))
	}))
	t.Cleanup(server.Close)

	_, err := connect.New(vendorhttp.WithBaseURL(server.URL)).Current(t.Context())

	apiErr, ok := errors.AsType[*connect.APIError](err)
	if !ok {
		t.Fatalf("Current() error = %v, want *APIError", err)
	}

	if apiErr.StatusCode != http.StatusNotFound || apiErr.Body != `{"message":"No prices"}` {
		t.Errorf("APIError = %+v, want 404 with the body", apiErr)
	}

	if !errors.Is(err, connect.ErrUnexpectedStatus) {
		t.Errorf("errors.Is(err, ErrUnexpectedStatus) = false for %v", err)
	}
}
