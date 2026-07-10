package sign

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"time"
)

// BuildDetachedCMS constructs a DER-encoded CMS SignedData detached signature.
func BuildDetachedCMS(digest []byte, signer Signer, signingTime time.Time, tsaToken []byte) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("%w: nil signer", ErrUnsupportedKey)
	}
	certs := signer.CertificateChain()
	if len(certs) == 0 {
		return nil, ErrNoCertificates
	}
	algo := signer.Algorithm()
	signedAttrs, err := buildSignedAttributes(digest, signingTime, certs[0], algo)
	if err != nil {
		return nil, err
	}
	signedAttrBytes, err := marshalAttributes(signedAttrs)
	if err != nil {
		return nil, err
	}
	h := algo.HashFunc().New()
	_, _ = h.Write(signedAttrBytes)
	attrDigest := h.Sum(nil)
	sig, err := signer.Sign(attrDigest)
	if err != nil {
		return nil, err
	}
	return marshalCMS(certs, algo, signedAttrBytes, sig, tsaToken)
}

func marshalCMS(certs []*x509.Certificate, algo Algorithm, signedAttrs, sig, tsaToken []byte) ([]byte, error) {
	certsLen := 0
	for _, cert := range certs {
		certsLen += len(cert.Raw)
	}
	certsDER := make([]byte, 0, certsLen)
	for _, cert := range certs {
		certsDER = append(certsDER, cert.Raw...)
	}
	digestAlgDER, err := asn1.Marshal(algorithmIdentifier{Algorithm: algo.DigestOID()})
	if err != nil {
		return nil, fmt.Errorf("sign: marshal digest algorithm: %w", err)
	}
	signerInfoDER, err := marshalSignerInfo(certs[0], algo, signedAttrs, sig, tsaToken)
	if err != nil {
		return nil, err
	}
	sd := signedData{
		Version:          1,
		DigestAlgorithms: asn1.RawValue{FullBytes: marshalSet(digestAlgDER)},
		EncapContentInfo: encapContentInfo{ContentType: oidData},
		Certificates:     asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: certsDER},
		SignerInfos:      asn1.RawValue{FullBytes: marshalSet(signerInfoDER)},
	}
	sdDER, err := asn1.Marshal(sd)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal signed data: %w", err)
	}
	ci := contentInfo{
		ContentType: oidSignedData,
		Content:     asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: sdDER},
	}
	out, err := asn1.Marshal(ci)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal content info: %w", err)
	}
	return out, nil
}

func buildSignedAttributes(digest []byte, signingTime time.Time, cert *x509.Certificate, algo Algorithm) ([]attribute, error) {
	contentTypeVal, err := asn1.Marshal(oidData)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal content type: %w", err)
	}
	digestVal, err := asn1.Marshal(asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagOctetString, Bytes: digest})
	if err != nil {
		return nil, fmt.Errorf("sign: marshal message digest: %w", err)
	}
	timeVal, err := asn1.Marshal(signingTime.UTC())
	if err != nil {
		return nil, fmt.Errorf("sign: marshal signing time: %w", err)
	}
	essDER, err := asn1.Marshal(signingCertificateV2{
		Certs: []essCertIDv2{{
			HashAlgorithm: algorithmIdentifier{Algorithm: algo.DigestOID()},
			CertHash:      hashBytes(algo.HashFunc(), cert.Raw),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("sign: marshal signing certificate: %w", err)
	}
	return []attribute{
		{Type: oidContentType, Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true, Bytes: contentTypeVal}},
		{Type: oidMessageDigest, Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true, Bytes: digestVal}},
		{Type: oidSigningTime, Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true, Bytes: timeVal}},
		{Type: oidSigningCertificateV2, Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true, Bytes: essDER}},
	}, nil
}

func marshalAttributes(attrs []attribute) ([]byte, error) {
	var attrsDER []byte
	for _, attr := range attrs {
		part, err := asn1.Marshal(attr)
		if err != nil {
			return nil, fmt.Errorf("sign: marshal attribute: %w", err)
		}
		attrsDER = append(attrsDER, part...)
	}
	return marshalSet(attrsDER), nil
}

func marshalSignerInfo(cert *x509.Certificate, algo Algorithm, signedAttrsSet, sig, tsaToken []byte) ([]byte, error) {
	serialDER, err := asn1.Marshal(cert.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal serial: %w", err)
	}
	info := signerInfo{
		Version: 1,
		SID: issuerAndSerialNumber{
			Issuer:       asn1.RawValue{FullBytes: cert.RawIssuer},
			SerialNumber: asn1.RawValue{FullBytes: serialDER},
		},
		DigestAlgorithm:    algorithmIdentifier{Algorithm: algo.DigestOID()},
		SignedAttrs:        asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: stripTag(signedAttrsSet)},
		SignatureAlgorithm: algorithmIdentifier{Algorithm: algo.SignatureOID()},
		Signature:          sig,
	}
	if len(tsaToken) > 0 {
		tsaDER, err := asn1.Marshal(attribute{
			Type:   oidTimeStampToken,
			Values: asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true, Bytes: tsaToken},
		})
		if err != nil {
			return nil, fmt.Errorf("sign: marshal timestamp attribute: %w", err)
		}
		info.UnsignedAttrs = asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 1, IsCompound: true, Bytes: tsaDER}
	}
	out, err := asn1.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("sign: marshal signer info: %w", err)
	}
	return out, nil
}

func marshalSet(content []byte) []byte {
	return marshalTLV(0x31, content)
}

func marshalTLV(tag byte, content []byte) []byte {
	length := len(content)
	if length < 128 {
		return append([]byte{tag, byte(length)}, content...)
	}
	lenBytes := lengthBytes(length)
	out := make([]byte, 0, 2+len(lenBytes)+len(content))
	out = append(out, tag)
	out = append(out, 0x80|byte(len(lenBytes))) // #nosec G115 -- lenBytes has at most the host int width.
	out = append(out, lenBytes...)
	return append(out, content...)
}

func lengthBytes(length int) []byte {
	var tmp []byte
	for length > 0 {
		tmp = append([]byte{byte(length & 0xff)}, tmp...)
		length >>= 8
	}
	return tmp
}

func stripTag(der []byte) []byte {
	if len(der) < 2 {
		return nil
	}
	lengthByte := der[1]
	if lengthByte&0x80 == 0 {
		return der[2:]
	}
	count := int(lengthByte & 0x7f)
	if len(der) < 2+count {
		return nil
	}
	return der[2+count:]
}

func hashBytes(hash crypto.Hash, data []byte) []byte {
	h := hash.New()
	_, _ = h.Write(data)
	return h.Sum(nil)
}
