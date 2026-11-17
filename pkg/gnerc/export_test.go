package gnerc

import "crypto/x509"

// Intermediate returns the embedded certificate, for a test that checks it.
func Intermediate() []byte {
	return intermediate
}

// RootsOver returns base with the embedded certificate added, as NewTransport builds its
// pool over the roots of the system.
func RootsOver(base *x509.CertPool) *x509.CertPool {
	return rootsOver(base)
}
