// PDF protection is adapted from the work of Klemen VODOPIVEC for the fpdf
// product.

package pdf

import (
	"crypto/md5" // #nosec G501 -- PDF standard security handler revision 2 uses MD5.
	"crypto/rand"
	"crypto/rc4" // #nosec G503 -- PDF standard security handler revision 2 uses RC4.
	"encoding/binary"
)

// Advisory bitflag constants that control document activities
const (
	CnProtectPrint      = 4
	CnProtectModify     = 8
	CnProtectCopy       = 16
	CnProtectAnnotForms = 32
)

type protectType struct {
	encrypted     bool
	uValue        []byte
	oValue        []byte
	pValue        int
	padding       []byte
	encryptionKey []byte
	objNum        int
	rc4cipher     *rc4.Cipher
	rc4n          uint32 // Object number associated with rc4 cipher
}

func (p *protectType) rc4(n uint32, buf *[]byte) {
	if p.rc4cipher == nil || p.rc4n != n {
		p.rc4cipher, _ = rc4.NewCipher(p.objectKey(n)) // #nosec G405 -- required by the PDF security handler.
		p.rc4n = n
	}
	p.rc4cipher.XORKeyStream(*buf, *buf)
}

func (p *protectType) objectKey(n uint32) []byte {
	nbuf := make([]byte, 8)
	binary.LittleEndian.PutUint32(nbuf, n)
	b := make([]byte, 0, len(p.encryptionKey)+5)
	b = append(b, p.encryptionKey...)
	b = append(b, nbuf[0], nbuf[1], nbuf[2], 0, 0)
	s := md5.Sum(b) // #nosec G401 -- required by the PDF security handler.
	return s[0:10]
}

func oValueGen(userPass, ownerPass []byte) []byte {
	var c *rc4.Cipher
	tmp := md5.Sum(ownerPass)        // #nosec G401 -- required by the PDF security handler.
	c, _ = rc4.NewCipher(tmp[0:5])   // #nosec G405 -- required by the PDF security handler.
	size := len(userPass)
	v := make([]byte, size)
	c.XORKeyStream(v, userPass)
	return v
}

func (p *protectType) uValueGen() []byte {
	var c *rc4.Cipher
	c, _ = rc4.NewCipher(p.encryptionKey) // #nosec G405 -- required by the PDF security handler.
	size := len(p.padding)
	v := make([]byte, size)
	c.XORKeyStream(v, p.padding)
	return v
}

func paddedProtectionPassword(pass, padding []byte) []byte {
	padded := make([]byte, 32)
	n := copy(padded, pass)
	if n < len(padded) {
		copy(padded[n:], padding)
	}
	return padded
}

func (p *protectType) setProtection(privFlag byte, userPassStr, ownerPassStr string) {
	privFlag = 192 | (privFlag & (CnProtectCopy | CnProtectModify | CnProtectPrint | CnProtectAnnotForms))
	p.padding = []byte{
		0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
		0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
		0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
		0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
	}
	userPass := []byte(userPassStr)
	var ownerPass []byte
	if ownerPassStr == "" {
		ownerPass = make([]byte, 8)
		if _, err := rand.Read(ownerPass); err != nil {
			copy(ownerPass, p.padding[:8])
		}
	} else {
		ownerPass = []byte(ownerPassStr)
	}
	userPass = paddedProtectionPassword(userPass, p.padding)
	ownerPass = paddedProtectionPassword(ownerPass, p.padding)
	p.encrypted = true
	p.oValue = oValueGen(userPass, ownerPass)
	buf := make([]byte, 0, len(userPass)+len(p.oValue)+4)
	buf = append(buf, userPass...)
	buf = append(buf, p.oValue...)
	buf = append(buf, privFlag, 0xff, 0xff, 0xff)
	sum := md5.Sum(buf) // #nosec G401 -- required by the PDF security handler.
	p.encryptionKey = sum[0:5]
	p.uValue = p.uValueGen()
	p.pValue = -(int(privFlag^255) + 1)
}
