package gulf_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/gulf"
)

// isGREASE tells whether a value is one of the reserved values of RFC 8701. A browser
// sends one. The handshake of Go never does.
func isGREASE(value uint16) bool {
	return value&0x0f0f == 0x0a0a && value>>8 == value&0xff
}

// GULFUEL-SC-003: The transport sends the hello of a browser, with http/1.1 as its only
// protocol.
func TestTransportSendsABrowserHello(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		hello *tls.ClientHelloInfo
	)

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	server := httptest.NewUnstartedServer(ok)
	server.TLS = &tls.Config{
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			mu.Lock()
			defer mu.Unlock()

			hello = info

			return nil, nil //nolint:nilnil // A nil config keeps the config of the server.
		},
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())

	client := &http.Client{Transport: gulf.NewTransportWithRoots(roots)}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK || resp.Proto != "HTTP/1.1" {
		t.Errorf("answer = %d %s, want 200 HTTP/1.1", resp.StatusCode, resp.Proto)
	}

	mu.Lock()
	defer mu.Unlock()

	if hello == nil {
		t.Fatal("the server saw no hello")
	}

	if !slices.Equal(hello.SupportedProtos, []string{"http/1.1"}) {
		t.Errorf("ALPN = %q, want [http/1.1]", hello.SupportedProtos)
	}

	if !slices.ContainsFunc(hello.CipherSuites, isGREASE) {
		t.Errorf("cipher suites %#04x hold no GREASE value", hello.CipherSuites)
	}
}
