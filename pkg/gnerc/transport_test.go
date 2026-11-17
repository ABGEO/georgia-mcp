package gnerc_test

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/gnerc"
)

const (
	intermediateSubject = "Go Daddy Secure Certificate Authority - G2"
	intermediateSHA256  = "973a41276ffd01e027a2aad49e34c37846d3e976ff6a620b6712e33832041aa6"
)

// GNERCMON-SC-005: The embedded certificate is the intermediate of GoDaddy, and the roots
// of NewTransport trust it.
func TestTransportTrustsTheIntermediate(t *testing.T) {
	t.Parallel()

	block, _ := pem.Decode(gnerc.Intermediate())
	if block == nil {
		t.Fatal("the embedded certificate is not PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(cert.Raw)
	if cert.Subject.CommonName != intermediateSubject ||
		hex.EncodeToString(sum[:]) != intermediateSHA256 {
		t.Errorf("certificate = %q %x, want %q %s",
			cert.Subject.CommonName, sum, intermediateSubject, intermediateSHA256)
	}

	roots := gnerc.RootsOver(x509.NewCertPool())
	if _, err := cert.Verify(x509.VerifyOptions{Roots: roots}); err != nil {
		t.Errorf("Verify() over the roots error = %v, want the certificate trusted", err)
	}

	transport := gnerc.NewTransport()
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil {
		t.Error("NewTransport() has no roots")
	}
}
