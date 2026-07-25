package sign_test

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/avdoseferovic/paper/pkg/reader"
	"github.com/avdoseferovic/paper/pkg/sign"
)

func TestNewLocalSignerRSA(t *testing.T) {
	t.Parallel()

	key, cert := testRSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}
	if signer.Algorithm() != sign.SHA256WithRSA {
		t.Fatalf("Algorithm() = %v, want SHA256WithRSA", signer.Algorithm())
	}
	digest := sha256.Sum256([]byte("payload"))
	sig, err := signer.Sign(digest[:])
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], sig); err != nil {
		t.Fatalf("signature verification error = %v", err)
	}
}

func TestNewLocalSignerECDSA(t *testing.T) {
	t.Parallel()

	key, cert := testECDSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}
	if signer.Algorithm() != sign.SHA256WithECDSA {
		t.Fatalf("Algorithm() = %v, want SHA256WithECDSA", signer.Algorithm())
	}
	digest := sha256.Sum256([]byte("payload"))
	sig, err := signer.Sign(digest[:])
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if !ecdsa.VerifyASN1(&key.PublicKey, digest[:], sig) {
		t.Fatal("ECDSA signature did not verify")
	}
}

func TestNewExternalSigner(t *testing.T) {
	t.Parallel()

	_, cert := testRSACertificate(t)
	signer, err := sign.NewExternalSigner(func(digest []byte) ([]byte, error) {
		return append([]byte("signed:"), digest...), nil
	}, []*x509.Certificate{cert}, sign.SHA384WithRSA)
	if err != nil {
		t.Fatalf("NewExternalSigner() error = %v", err)
	}

	sig, err := signer.Sign([]byte("digest"))
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if string(sig) != "signed:digest" {
		t.Fatalf("Sign() = %q", sig)
	}
	if signer.Algorithm().HashFunc() != crypto.SHA384 {
		t.Fatalf("HashFunc() = %v, want SHA384", signer.Algorithm().HashFunc())
	}
}

func TestBuildDetachedCMS(t *testing.T) {
	t.Parallel()

	key, cert := testRSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}
	digest := sha256.Sum256([]byte("byte ranges"))

	cms, err := sign.BuildDetachedCMS(digest[:], signer, time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("BuildDetachedCMS() error = %v", err)
	}
	if len(cms) == 0 {
		t.Fatal("BuildDetachedCMS() returned empty output")
	}

	var ci struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}
	if _, err := asn1.Unmarshal(cms, &ci); err != nil {
		t.Fatalf("CMS ASN.1 unmarshal error = %v", err)
	}
	if !ci.ContentType.Equal(sign.OIDSignedData()) {
		t.Fatalf("ContentType = %v, want signedData", ci.ContentType)
	}
	if !bytes.Contains(cms, cert.Raw) {
		t.Fatal("CMS output should embed the signing certificate")
	}
}

func TestPDFBB(t *testing.T) {
	t.Parallel()

	key, cert := testRSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}
	pdf := minimalPDF(t)

	signed, err := sign.PDF(pdf, sign.Options{
		Signer:      signer,
		Level:       sign.LevelBB,
		Name:        "Paper Test Signer",
		Reason:      "Unit test",
		Location:    "Test Lab",
		SigningTime: time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PDF() error = %v", err)
	}
	if len(signed) <= len(pdf) {
		t.Fatalf("signed PDF length = %d, want > %d", len(signed), len(pdf))
	}
	if !bytes.HasPrefix(signed, pdf) {
		t.Fatal("signed PDF should preserve the original PDF as an incremental update")
	}
	for _, marker := range [][]byte{
		[]byte("/Type /Sig"),
		[]byte("/SubFilter /ETSI.CAdES.detached"),
		[]byte("/ByteRange"),
		[]byte("/AcroForm"),
		[]byte("/Prev"),
	} {
		if !bytes.Contains(signed, marker) {
			t.Fatalf("signed PDF missing marker %q", marker)
		}
	}
	assertContentsPatched(t, signed)

	parsed, err := reader.Parse(signed)
	if err != nil {
		t.Fatalf("signed PDF should remain parseable: %v", err)
	}
	if parsed.PageCount() != 1 {
		t.Fatalf("PageCount() = %d, want 1", parsed.PageCount())
	}
}

func TestPDFECDSA(t *testing.T) {
	t.Parallel()

	key, cert := testECDSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}

	signed, err := sign.PDF(minimalPDF(t), sign.Options{Signer: signer})
	if err != nil {
		t.Fatalf("PDF() error = %v", err)
	}
	if !bytes.Contains(signed, []byte("/Type /Sig")) {
		t.Fatal("signed PDF missing /Type /Sig")
	}
	assertContentsPatched(t, signed)
}

func TestPDFNilSigner(t *testing.T) {
	t.Parallel()

	_, err := sign.PDF(minimalPDF(t), sign.Options{})
	if err == nil {
		t.Fatal("PDF() expected nil signer error")
	}
}

func TestPDFBTRequiresTSA(t *testing.T) {
	t.Parallel()

	key, cert := testRSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}
	_, err = sign.PDF(minimalPDF(t), sign.Options{Signer: signer, Level: sign.LevelBT})
	if err == nil {
		t.Fatal("PDF() expected B-T without TSA error")
	}
}

func TestPDFBTEmbedsTSAToken(t *testing.T) {
	t.Parallel()

	key, cert := testRSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}
	token := []byte{0x30, 0x03, 0x02, 0x01, 0x01}
	response := timestampResponse(t, token)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/timestamp-query" {
			t.Errorf("Content-Type = %q, want application/timestamp-query", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(response)
	}))
	defer server.Close()

	signed, err := sign.PDF(minimalPDF(t), sign.Options{
		Signer:    signer,
		Level:     sign.LevelBT,
		TSAClient: sign.NewTSAClient(server.URL),
	})
	if err != nil {
		t.Fatalf("PDF() error = %v", err)
	}
	assertContentsPatched(t, signed)
}

func TestAddDSS(t *testing.T) {
	t.Parallel()

	dss := sign.NewDSS()
	dss.AddSignatureValidation(
		[]byte("signature"),
		[]*x509.Certificate{{Raw: []byte("cert-der")}},
		[][]byte{[]byte("ocsp-der")},
		[][]byte{[]byte("crl-der")},
	)

	out, err := sign.AddDSS(minimalPDF(t), dss)
	if err != nil {
		t.Fatalf("AddDSS() error = %v", err)
	}
	for _, marker := range [][]byte{
		[]byte("/DSS"),
		[]byte("/Certs"),
		[]byte("/OCSPs"),
		[]byte("/CRLs"),
		[]byte("/VRI"),
	} {
		if !bytes.Contains(out, marker) {
			t.Fatalf("DSS output missing marker %q", marker)
		}
	}
	if _, err := reader.Parse(out); err != nil {
		t.Fatalf("DSS output should remain parseable: %v", err)
	}
}

func TestOCSPClientFetchResponse(t *testing.T) {
	t.Parallel()

	issuerKey, issuer := testRSACertificate(t)
	response := ocspResponse(t, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/ocsp-request" {
			t.Errorf("Content-Type = %q, want application/ocsp-request", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/ocsp-response")
		_, _ = w.Write(response)
	}))
	defer server.Close()

	leaf := testLeafCertificate(t, issuerKey, issuer, server.URL)
	got, err := sign.NewOCSPClient().FetchResponse(leaf, issuer)
	if err != nil {
		t.Fatalf("FetchResponse() error = %v", err)
	}
	if !bytes.Equal(got, response) {
		t.Fatal("FetchResponse() returned unexpected response bytes")
	}
}

func TestPDFBLTEmbedsDSS(t *testing.T) {
	t.Parallel()

	key, cert := testRSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}

	signed, err := sign.PDF(minimalPDF(t), sign.Options{
		Signer:    signer,
		Level:     sign.LevelBLT,
		TSAClient: testTSAClient(t),
		CRLs:      [][]byte{[]byte("crl-der")},
		ExtraCerts: [][]byte{
			[]byte("extra-cert-der"),
		},
	})
	if err != nil {
		t.Fatalf("PDF() error = %v", err)
	}
	for _, marker := range [][]byte{[]byte("/DSS"), []byte("/VRI"), []byte("/CRLs"), []byte("/Certs")} {
		if !bytes.Contains(signed, marker) {
			t.Fatalf("B-LT signed PDF missing marker %q", marker)
		}
	}
}

func TestPDFBLTAAddsDocumentTimestamp(t *testing.T) {
	t.Parallel()

	key, cert := testRSACertificate(t)
	signer, err := sign.NewLocalSigner(key, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("NewLocalSigner() error = %v", err)
	}

	signed, err := sign.PDF(minimalPDF(t), sign.Options{
		Signer:    signer,
		Level:     sign.LevelBLTA,
		TSAClient: testTSAClient(t),
	})
	if err != nil {
		t.Fatalf("PDF() error = %v", err)
	}
	if !bytes.Contains(signed, []byte("/Type /DocTimeStamp")) {
		t.Fatal("B-LTA signed PDF missing document timestamp")
	}
}

func testRSACertificate(t *testing.T) (*rsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	cert := selfSignedCertificate(t, key)
	return key, cert
}

func testECDSACertificate(t *testing.T) (*ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	cert := selfSignedCertificate(t, key)
	return key, cert
}

func selfSignedCertificate(t *testing.T, key crypto.Signer) *x509.Certificate {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Paper Test Signer"},
		NotBefore:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	return cert
}

func testLeafCertificate(t *testing.T, issuerKey *rsa.PrivateKey, issuer *x509.Certificate, ocspURL string) *x509.Certificate {
	t.Helper()
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Paper OCSP Leaf"},
		NotBefore:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		OCSPServer:   []string{ocspURL},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, leafKey.Public(), issuerKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	return cert
}

func minimalPDF(t *testing.T) []byte {
	t.Helper()
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>",
		"<< /Length 0 >>\nstream\n\nendstream",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, 0, len(objects))
	for i, object := range objects {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return buf.Bytes()
}

func testTSAClient(t *testing.T) *sign.TSAClient {
	t.Helper()
	response := timestampResponse(t, []byte{0x30, 0x03, 0x02, 0x01, 0x01})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(response)
	}))
	t.Cleanup(server.Close)
	return sign.NewTSAClient(server.URL)
}

func assertContentsPatched(t *testing.T, signed []byte) {
	t.Helper()
	contentsIdx := bytes.Index(signed, []byte("/Contents <"))
	if contentsIdx < 0 {
		t.Fatal("signed PDF missing /Contents")
	}
	hexStart := contentsIdx + len("/Contents <")
	hexArea := signed[hexStart : hexStart+16]
	if bytes.Equal(hexArea, []byte("0000000000000000")) {
		t.Fatal("/Contents appears to be unpatched")
	}
}

func timestampResponse(t *testing.T, token []byte) []byte {
	t.Helper()
	resp := struct {
		Status struct {
			Status int
		}
		TimeStampToken asn1.RawValue `asn1:"optional"`
	}{
		TimeStampToken: asn1.RawValue{FullBytes: token},
	}
	der, err := asn1.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal timestamp response: %v", err)
	}
	if len(der) == 0 {
		t.Fatal("empty timestamp response: " + strconv.Itoa(len(der)))
	}
	return der
}

func ocspResponse(t *testing.T, status asn1.Enumerated) []byte {
	t.Helper()
	resp := struct {
		ResponseStatus asn1.Enumerated
	}{ResponseStatus: status}
	der, err := asn1.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal OCSP response: %v", err)
	}
	return der
}
