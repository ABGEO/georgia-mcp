package gnerc

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"net/http"
)

// intermediate is the certificate Go Daddy Secure Certificate Authority - G2. The server
// of GNERC sends its own certificate without it, and Go fetches no missing certificate.
//
//go:embed godaddy-g2.pem
var intermediate []byte

// NewTransport returns a transport that trusts the roots of the system and the
// intermediate certificate that GNERC does not send. The default client of New uses it.
// An HTTP client that a caller passes with vendorhttp.WithHTTPClient needs it too, or
// the handshake fails on a system that does not fetch the certificate.
func NewTransport() *http.Transport {
	base, err := x509.SystemCertPool()
	if err != nil {
		base = x509.NewCertPool()
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		transport = &http.Transport{}
	}

	transport = transport.Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: rootsOver(base), MinVersion: tls.VersionTLS12}

	return transport
}

func rootsOver(base *x509.CertPool) *x509.CertPool {
	base.AppendCertsFromPEM(intermediate)

	return base
}
