package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
)

// Errors reported when a signer cannot be built from the given key material.
var (
	ErrNoCertificates    = errors.New("sign: no certificates provided")
	ErrNilSignFunction   = errors.New("sign: signFn must not be nil")
	ErrUnsupportedKey    = errors.New("sign: unsupported key type")
	ErrPKCS12Unsupported = errors.New(
		"sign: PKCS12 loading requires golang.org/x/crypto/pkcs12; use NewLocalSigner with pre-parsed key and certificate",
	)
)

// Signer performs a cryptographic signing operation.
type Signer interface {
	Sign(digest []byte) ([]byte, error)
	Algorithm() Algorithm
	CertificateChain() []*x509.Certificate
}

// LocalSigner signs with a local private key.
type LocalSigner struct {
	key   crypto.Signer
	certs []*x509.Certificate
	algo  Algorithm
}

// NewLocalSigner creates a signer from a local RSA or ECDSA private key.
func NewLocalSigner(key crypto.Signer, certs []*x509.Certificate) (*LocalSigner, error) {
	if len(certs) == 0 {
		return nil, ErrNoCertificates
	}
	var algo Algorithm
	switch key.(type) {
	case *rsa.PrivateKey:
		algo = SHA256WithRSA
	case *ecdsa.PrivateKey:
		algo = SHA256WithECDSA
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedKey, key)
	}
	return &LocalSigner{key: key, certs: append([]*x509.Certificate(nil), certs...), algo: algo}, nil
}

// SetAlgorithm overrides the auto-detected algorithm.
func (s *LocalSigner) SetAlgorithm(algo Algorithm) {
	s.algo = algo
}

// Sign signs digest with the configured local key.
func (s *LocalSigner) Sign(digest []byte) ([]byte, error) {
	sig, err := s.key.Sign(rand.Reader, digest, s.algo.HashFunc())
	if err != nil {
		return nil, fmt.Errorf("sign: local signer: %w", err)
	}
	return sig, nil
}

// Algorithm returns the signature algorithm.
func (s *LocalSigner) Algorithm() Algorithm { return s.algo }

// CertificateChain returns a copy of the certificate chain.
func (s *LocalSigner) CertificateChain() []*x509.Certificate {
	return append([]*x509.Certificate(nil), s.certs...)
}

// ExternalSigner delegates signing to a caller-supplied function.
type ExternalSigner struct {
	signFn func(digest []byte) ([]byte, error)
	certs  []*x509.Certificate
	algo   Algorithm
}

// NewExternalSigner creates a signer backed by an external signing function.
func NewExternalSigner(
	signFn func(digest []byte) ([]byte, error),
	certs []*x509.Certificate,
	algo Algorithm,
) (*ExternalSigner, error) {
	if signFn == nil {
		return nil, ErrNilSignFunction
	}
	if len(certs) == 0 {
		return nil, ErrNoCertificates
	}
	return &ExternalSigner{signFn: signFn, certs: append([]*x509.Certificate(nil), certs...), algo: algo}, nil
}

// Sign delegates digest signing to the configured function.
func (s *ExternalSigner) Sign(digest []byte) ([]byte, error) {
	sig, err := s.signFn(digest)
	if err != nil {
		return nil, fmt.Errorf("sign: external signer: %w", err)
	}
	return sig, nil
}

// Algorithm returns the signature algorithm.
func (s *ExternalSigner) Algorithm() Algorithm { return s.algo }

// CertificateChain returns a copy of the certificate chain.
func (s *ExternalSigner) CertificateChain() []*x509.Certificate {
	return append([]*x509.Certificate(nil), s.certs...)
}

// ParsePKCS12 is reserved for future PKCS#12 support.
func ParsePKCS12(_ []byte, _ string) (*LocalSigner, error) {
	return nil, ErrPKCS12Unsupported
}
