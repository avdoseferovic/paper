// Package sign provides signing primitives used by PDF signature workflows.
package sign

import (
	"crypto"
	"encoding/asn1"
)

var (
	oidData       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}

	oidContentType          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidMessageDigest        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	oidSigningTime          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}
	oidSigningCertificateV2 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 47}
	oidTimeStampToken       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14}

	oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidSHA384 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	oidSHA512 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}

	oidSHA256WithRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidSHA384WithRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	oidSHA512WithRSA = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}

	oidECDSAWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidECDSAWithSHA384 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	oidECDSAWithSHA512 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}
)

// OIDSignedData returns the CMS signedData object identifier.
func OIDSignedData() asn1.ObjectIdentifier {
	return append(asn1.ObjectIdentifier(nil), oidSignedData...)
}

// Algorithm identifies a hash and signature algorithm pair.
type Algorithm int

// The hash and signature algorithm pairs available for signing.
const (
	SHA256WithRSA Algorithm = iota
	SHA384WithRSA
	SHA512WithRSA
	SHA256WithECDSA
	SHA384WithECDSA
	SHA512WithECDSA
)

// HashFunc returns the hash used by the algorithm.
func (a Algorithm) HashFunc() crypto.Hash {
	switch a {
	case SHA256WithRSA, SHA256WithECDSA:
		return crypto.SHA256
	case SHA384WithRSA, SHA384WithECDSA:
		return crypto.SHA384
	case SHA512WithRSA, SHA512WithECDSA:
		return crypto.SHA512
	default:
		return crypto.SHA256
	}
}

// DigestOID returns the digest algorithm OID.
func (a Algorithm) DigestOID() asn1.ObjectIdentifier {
	switch a {
	case SHA256WithRSA, SHA256WithECDSA:
		return oidSHA256
	case SHA384WithRSA, SHA384WithECDSA:
		return oidSHA384
	case SHA512WithRSA, SHA512WithECDSA:
		return oidSHA512
	default:
		return oidSHA256
	}
}

// SignatureOID returns the signature algorithm OID.
func (a Algorithm) SignatureOID() asn1.ObjectIdentifier {
	switch a {
	case SHA256WithRSA:
		return oidSHA256WithRSA
	case SHA384WithRSA:
		return oidSHA384WithRSA
	case SHA512WithRSA:
		return oidSHA512WithRSA
	case SHA256WithECDSA:
		return oidECDSAWithSHA256
	case SHA384WithECDSA:
		return oidECDSAWithSHA384
	case SHA512WithECDSA:
		return oidECDSAWithSHA512
	default:
		return oidSHA256WithRSA
	}
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,tag:0"`
}

type signedData struct {
	Version          int
	DigestAlgorithms asn1.RawValue
	EncapContentInfo encapContentInfo
	Certificates     asn1.RawValue `asn1:"optional,tag:0"`
	SignerInfos      asn1.RawValue
}

type encapContentInfo struct {
	ContentType asn1.ObjectIdentifier
}

type signerInfo struct {
	Version            int
	SID                issuerAndSerialNumber
	DigestAlgorithm    algorithmIdentifier
	SignedAttrs        asn1.RawValue `asn1:"optional,tag:0"`
	SignatureAlgorithm algorithmIdentifier
	Signature          []byte
	UnsignedAttrs      asn1.RawValue `asn1:"optional,tag:1"`
}

type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber asn1.RawValue
}

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type attribute struct {
	Type   asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}

type essCertIDv2 struct {
	HashAlgorithm algorithmIdentifier `asn1:"optional"`
	CertHash      []byte
}

type signingCertificateV2 struct {
	Certs []essCertIDv2
}
