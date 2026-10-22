package gulf

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"time"

	utls "github.com/refraction-networking/utls"
)

const (
	dialTimeout     = 10 * time.Second
	keepAlive       = 30 * time.Second
	idleConnTimeout = 90 * time.Second
)

// NewTransport returns a transport that dials each TLS connection with the handshake of
// Chrome 131, over HTTP/1.1. The bot filter of Gulf refuses the handshake of Go. The
// transport uses no proxy.
func NewTransport() *http.Transport {
	return newTransport(nil)
}

// newTransport verifies the certificate of the server with roots, or with the roots of
// the system when roots is nil.
func newTransport(roots *x509.CertPool) *http.Transport {
	dialer := &net.Dialer{Timeout: dialTimeout, KeepAlive: keepAlive}

	return &http.Transport{
		DialContext: dialer.DialContext,
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialBrowserTLS(ctx, dialer, network, addr, roots)
		},
		IdleConnTimeout: idleConnTimeout,
	}
}

func dialBrowserTLS(
	ctx context.Context,
	dialer *net.Dialer,
	network, addr string,
	roots *x509.CertPool,
) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("reading the address %q: %w", addr, err)
	}

	raw, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", addr, err)
	}

	conn := utls.UClient(raw, &utls.Config{ServerName: host, RootCAs: roots}, utls.HelloChrome_131)

	if err := conn.BuildHandshakeState(); err != nil {
		_ = raw.Close()

		return nil, fmt.Errorf("building the hello: %w", err)
	}

	// net/http speaks HTTP/2 only over a *tls.Conn. If the hello offers h2, the server
	// selects it and the transport cannot read the answer.
	for _, extension := range conn.Extensions {
		if alpn, ok := extension.(*utls.ALPNExtension); ok {
			alpn.AlpnProtocols = []string{"http/1.1"}
		}
	}

	if err := conn.HandshakeContext(ctx); err != nil {
		_ = raw.Close()

		return nil, fmt.Errorf("the TLS handshake with %s: %w", addr, err)
	}

	return conn, nil
}
