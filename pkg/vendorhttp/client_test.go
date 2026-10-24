package vendorhttp_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/vendorhttp"
)

var errUnexpectedStatus = errors.New("acme: unexpected status")

// recorded is the last request that the server got.
type recorded struct {
	method string
	uri    string
	header http.Header
	body   string
}

func serve(t *testing.T, status int, body string) (*httptest.Server, *recorded) {
	t.Helper()

	got := &recorded{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent, _ := io.ReadAll(r.Body)
		*got = recorded{
			method: r.Method,
			uri:    r.URL.RequestURI(),
			header: r.Header.Clone(),
			body:   string(sent),
		}

		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server, got
}

func client(server *httptest.Server) *vendorhttp.Client {
	return vendorhttp.New(errUnexpectedStatus,
		vendorhttp.WithBaseURL(server.URL),
		vendorhttp.WithHeader("Accept", "application/json"),
		vendorhttp.WithMaxBody(8),
	)
}

// PLATFORM-SC-037: Fetch sends the path, the query, the header of the client, and the
// User-Agent of the server, and returns the body of a 200.
func TestFetchSendsTheRequest(t *testing.T) {
	t.Parallel()

	server, got := serve(t, http.StatusOK, `{"x":1}`)

	body, err := client(server).Fetch(t.Context(), vendorhttp.Request{
		Path:  []string{"rates"},
		Query: url.Values{"a": {"1"}},
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if string(body) != `{"x":1}` {
		t.Errorf("body = %q, want {\"x\":1}", body)
	}

	if got.method != http.MethodGet || got.uri != "/rates?a=1" {
		t.Errorf("request = %s %s, want GET /rates?a=1", got.method, got.uri)
	}

	if accept := got.header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	agent := got.header.Get("User-Agent")
	if !strings.HasPrefix(agent, "georgia-mcp/") ||
		!strings.HasSuffix(agent, "(+https://github.com/abgeo/georgia-mcp)") {
		t.Errorf("User-Agent = %q, want georgia-mcp/<version> (+<repository>)", agent)
	}
}

// PLATFORM-SC-037: The client reads at most the limit of the body.
func TestFetchLimitsTheBody(t *testing.T) {
	t.Parallel()

	server, _ := serve(t, http.StatusOK, strings.Repeat("a", 20))

	body, err := client(server).Fetch(t.Context(), vendorhttp.Request{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(body) != 8 {
		t.Errorf("body holds %d bytes, want 8", len(body))
	}
}

// PLATFORM-SC-037: A status other than 200 is a StatusError that wraps the sentinel of
// the client.
func TestFetchReportsAStatus(t *testing.T) {
	t.Parallel()

	server, _ := serve(t, http.StatusServiceUnavailable, "down")

	_, err := client(server).Fetch(t.Context(), vendorhttp.Request{})

	statusErr, ok := errors.AsType[*vendorhttp.StatusError](err)
	if !ok || statusErr.StatusCode != http.StatusServiceUnavailable || statusErr.Body != "down" {
		t.Fatalf("Fetch() error = %v, want a *StatusError with 503 and the body", err)
	}

	if !errors.Is(err, errUnexpectedStatus) {
		t.Errorf("errors.Is(err, the sentinel) = false for %v", err)
	}

	want := "acme: unexpected status: 503: Service Unavailable"
	if statusErr.Error() != want {
		t.Errorf("Error() = %q, want %q", statusErr.Error(), want)
	}
}

// PLATFORM-SC-037: Do returns the answer of any status, with no error.
func TestDoReturnsAnyStatus(t *testing.T) {
	t.Parallel()

	server, _ := serve(t, http.StatusNoContent, "")

	resp, err := client(server).Do(t.Context(), vendorhttp.Request{})
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Do() = %+v, %v, want 204 and no error", resp, err)
	}
}

// PLATFORM-SC-037: FetchJSON decodes the body into the target.
func TestFetchJSONDecodesTheBody(t *testing.T) {
	t.Parallel()

	server, _ := serve(t, http.StatusOK, `{"x":1}`)

	var target struct {
		X int `json:"x"`
	}

	if err := client(server).FetchJSON(t.Context(), vendorhttp.Request{}, &target); err != nil {
		t.Fatalf("FetchJSON() error = %v", err)
	}

	if target.X != 1 {
		t.Errorf("X = %d, want 1", target.X)
	}
}

// PLATFORM-SC-037: A request sends its method and its body, and its header wins over the
// header of the client.
func TestDoSendsTheMethodTheBodyAndTheHeader(t *testing.T) {
	t.Parallel()

	server, got := serve(t, http.StatusOK, "")

	_, err := client(server).Do(t.Context(), vendorhttp.Request{
		Method: http.MethodPost,
		Header: http.Header{"Accept": {"text/html"}},
		Body:   []byte(`{"q":1}`),
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if got.method != http.MethodPost || got.body != `{"q":1}` {
		t.Errorf("request = %s %q, want POST {\"q\":1}", got.method, got.body)
	}

	if accept := got.header.Get("Accept"); accept != "text/html" {
		t.Errorf("Accept = %q, want text/html", accept)
	}
}
