package gulf

import (
	"crypto/x509"
	"net/http"
)

// NewTransportWithRoots builds the transport of NewTransport over roots, for a test that
// trusts the certificate of a local server.
func NewTransportWithRoots(roots *x509.CertPool) *http.Transport {
	return newTransport(roots)
}
