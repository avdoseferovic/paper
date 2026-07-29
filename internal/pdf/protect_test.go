package pdf

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rc4"
	"regexp"
	"slices"
	"strconv"
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
	if err := p.setProtection(CnProtectCopy, "user", "owner"); err != nil {
		t.Fatalf("setProtection: %v", err)
	}

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
	decrypted := slices.Clone(encrypted[aes.BlockSize:])
	cipher.NewCBCDecrypter(block, encrypted[:aes.BlockSize]).CryptBlocks(decrypted, decrypted)
	unpadded, ok := stripPKCS7(decrypted)
	if !ok {
		t.Fatalf("invalid PKCS#7 padding in %x", decrypted)
	}
	if !bytes.Equal(unpadded, plain) {
		t.Fatalf("expected decrypted %q, got %q", plain, unpadded)
	}
}

func TestProtectionRC4EncryptsEachStringIndependently(t *testing.T) {
	t.Parallel()

	var p protectType
	if err := p.setProtection(CnProtectCopy, "user", "owner"); err != nil {
		t.Fatalf("setProtection: %v", err)
	}

	const objNum = 7
	first, err := p.encryptBytes(objNum, []byte("first string"))
	if err != nil {
		t.Fatalf("encrypt first: %v", err)
	}
	second, err := p.encryptBytes(objNum, []byte("second string"))
	if err != nil {
		t.Fatalf("encrypt second: %v", err)
	}

	if got := rc4DecryptWithObjectKey(t, &p, objNum, first); got != "first string" {
		t.Fatalf("first string decrypted to %q", got)
	}
	if got := rc4DecryptWithObjectKey(t, &p, objNum, second); got != "second string" {
		t.Fatalf("second string decrypted to %q; RC4 keystream must restart per string", got)
	}
}

func TestProtectionRC4InfoStringsRoundTrip(t *testing.T) {
	t.Parallel()

	f := newProtectionTestPDF()
	f.SetTitle("Secret Title", false)
	f.SetAuthor("Secret Author", false)
	f.SetProtection(CnProtectCopy, "user", "owner")

	var out bytes.Buffer
	if err := f.Output(&out); err != nil {
		t.Fatalf("output protected pdf: %v", err)
	}

	pdf := out.Bytes()
	infoMatch := regexp.MustCompile(`/Info (\d+) 0 R`).FindSubmatch(pdf)
	if infoMatch == nil {
		t.Fatal("missing /Info reference in trailer")
	}
	infoNum, err := strconv.Atoi(string(infoMatch[1]))
	if err != nil {
		t.Fatalf("parse info object number: %v", err)
	}
	objStart := bytes.Index(pdf, []byte("\n"+string(infoMatch[1])+" 0 obj"))
	if objStart < 0 {
		t.Fatalf("info object %d not found", infoNum)
	}
	infoObj := pdf[objStart:]
	if end := bytes.Index(infoObj, []byte("endobj")); end >= 0 {
		infoObj = infoObj[:end]
	}

	objNum, ok := checkedUint32(infoNum)
	if !ok {
		t.Fatalf("info object number out of range: %d", infoNum)
	}
	title := rc4DecryptWithObjectKey(t, &f.protect, objNum, extractPDFLiteralString(t, infoObj, "/Title "))
	if title != "Secret Title" {
		t.Fatalf("decrypted /Title = %q", title)
	}
	author := rc4DecryptWithObjectKey(t, &f.protect, objNum, extractPDFLiteralString(t, infoObj, "/Author "))
	if author != "Secret Author" {
		t.Fatalf("decrypted /Author = %q; each Info string must use a fresh RC4 keystream", author)
	}
}

// extractPDFLiteralString returns the unescaped bytes of the literal string
// that follows key, e.g. `/Title (...)`.
func extractPDFLiteralString(t *testing.T, obj []byte, key string) []byte {
	t.Helper()
	start := bytes.Index(obj, []byte(key+"("))
	if start < 0 {
		t.Fatalf("missing %s(...) entry", key)
	}
	raw := obj[start+len(key)+1:]
	var value []byte
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c == '\\' && i+1 < len(raw) {
			i++
			switch raw[i] {
			case 'r':
				value = append(value, '\r')
			default:
				value = append(value, raw[i])
			}
			continue
		}
		if c == ')' {
			return value
		}
		value = append(value, c)
	}
	t.Fatalf("unterminated literal string after %s", key)
	return nil
}

func rc4DecryptWithObjectKey(t *testing.T, p *protectType, n uint32, data []byte) string {
	t.Helper()
	c, err := rc4.NewCipher(p.objectKey(n))
	if err != nil {
		t.Fatalf("rc4 cipher: %v", err)
	}
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return string(out)
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

// TestSetProtectionFailsWhenRandomSourceFails covers a random source that cannot
// supply the generated owner password or file ID. Both used to fall back to the
// PDF spec's published padding string, producing a document that looked
// encrypted while an attacker knew every generated input to key derivation.
func TestSetProtectionFailsWhenRandomSourceFails(t *testing.T) {
	t.Parallel()

	for name, algorithm := range map[string]ProtectionAlgorithm{
		"rc4":     ProtectionRC4,
		"aes-128": ProtectionAES128,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newProtectionTestPDF()
			f.SetProtectionAlgorithm(algorithm)
			f.protect.random = bytes.NewReader(nil)

			f.SetProtection(CnProtectCopy, "", "")

			if f.Error() == nil {
				t.Fatal("SetProtection() error = nil, want a random source failure")
			}
			if f.protect.encrypted {
				t.Fatal("document is marked encrypted after a failed key setup")
			}
		})
	}
}

// TestSetProtectionWithOwnerPasswordNeedsNoRandomness documents that only the
// generated owner password and file ID need the random source.
func TestSetProtectionWithOwnerPasswordNeedsNoRandomness(t *testing.T) {
	t.Parallel()

	f := newProtectionTestPDF()
	f.protect.random = bytes.NewReader(nil)

	f.SetProtection(CnProtectCopy, "user", "owner")

	if f.Error() != nil {
		t.Fatalf("SetProtection() error = %v, want nil for RC4 with an explicit owner password", f.Error())
	}
	if !f.protect.encrypted {
		t.Fatal("document is not marked encrypted")
	}
}
