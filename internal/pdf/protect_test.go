package pdf

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"strings"
	"testing"
)

func TestProtectionRC4KeepsLegacyDictionary(t *testing.T) {
	t.Parallel()

	f := newProtectionTestPDF()
	f.SetProtection(CnProtectCopy, "user", "owner")

	var out bytes.Buffer
	if err := f.Output(&out); err != nil {
		t.Fatalf("output protected pdf: %v", err)
	}

	pdf := out.String()
	for _, want := range []string{
		"/Filter /Standard",
		"/V 1",
		"/R 2",
		"/ID [()()]",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("expected legacy protected PDF to contain %q", want)
		}
	}
	if strings.Contains(pdf, "/AESV2") {
		t.Fatal("legacy protected PDF unexpectedly used AESV2")
	}
}

func TestProtectionAES128WritesRevision4Dictionary(t *testing.T) {
	t.Parallel()

	f := newProtectionTestPDF()
	f.SetProtectionAlgorithm(ProtectionAES128)
	f.protect.random = bytes.NewReader(bytes.Repeat([]byte{0x3a}, 4096))
	f.SetProtection(CnProtectCopy, "user", "owner")

	var out bytes.Buffer
	if err := f.Output(&out); err != nil {
		t.Fatalf("output protected pdf: %v", err)
	}

	pdf := out.String()
	for _, want := range []string{
		"/Filter /Standard",
		"/V 4",
		"/R 4",
		"/Length 128",
		"/CF <</StdCF <</CFM /AESV2 /AuthEvent /DocOpen /Length 128>>>>",
		"/StmF /StdCF",
		"/StrF /StdCF",
		"/ID [<3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a><3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a>]",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("expected AES protected PDF to contain %q", want)
		}
	}
}

func TestProtectionAES128EncryptsWithIVAndPKCS7Padding(t *testing.T) {
	t.Parallel()

	random := bytes.NewReader(append(
		bytes.Repeat([]byte{0x11}, aes.BlockSize),
		bytes.Repeat([]byte{0x22}, aes.BlockSize)...,
	))
	p := protectType{
		algorithm: ProtectionAES128,
		random:    random,
	}
	p.setProtection(CnProtectCopy, "user", "owner")

	plain := []byte("hello")
	encrypted, err := p.encryptBytes(7, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if len(encrypted) != aes.BlockSize*2 {
		t.Fatalf("expected IV plus one AES block, got %d bytes", len(encrypted))
	}
	if !bytes.Equal(encrypted[:aes.BlockSize], bytes.Repeat([]byte{0x22}, aes.BlockSize)) {
		t.Fatalf("unexpected IV %x", encrypted[:aes.BlockSize])
	}

	block, err := aes.NewCipher(p.aesObjectKey(7))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	decrypted := append([]byte(nil), encrypted[aes.BlockSize:]...)
	cipher.NewCBCDecrypter(block, encrypted[:aes.BlockSize]).CryptBlocks(decrypted, decrypted)
	unpadded, ok := stripPKCS7(decrypted)
	if !ok {
		t.Fatalf("invalid PKCS#7 padding in %x", decrypted)
	}
	if !bytes.Equal(unpadded, plain) {
		t.Fatalf("expected decrypted %q, got %q", plain, unpadded)
	}
}

func newProtectionTestPDF() *PDF {
	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	f.AddPage()
	f.SetFont("Arial", "", 12)
	f.Text(10, 10, "hello")
	return f
}

func stripPKCS7(data []byte) ([]byte, bool) {
	if len(data) == 0 {
		return nil, false
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > aes.BlockSize || padLen > len(data) {
		return nil, false
	}
	for _, b := range data[len(data)-padLen:] {
		if int(b) != padLen {
			return nil, false
		}
	}
	return data[:len(data)-padLen], true
}
