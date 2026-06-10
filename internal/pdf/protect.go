// PDF protection is adapted from the work of Klemen VODOPIVEC for the fpdf
// product.

package pdf

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" // #nosec G501 -- PDF standard security handler revision 2 uses MD5.
	cryptoRand "crypto/rand"
	"crypto/rc4" // #nosec G503 -- PDF standard security handler revision 2 uses RC4.
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Advisory bitflag constants that control document activities
const (
	CnProtectPrint      = 4
	CnProtectModify     = 8
	CnProtectCopy       = 16
	CnProtectAnnotForms = 32
)

// ProtectionAlgorithm selects the PDF standard security handler encryption
// algorithm used by protected documents.
type ProtectionAlgorithm byte

const (
	// ProtectionRC4 keeps the legacy RC4 protection behavior.
	ProtectionRC4 ProtectionAlgorithm = iota
	// ProtectionAES128 selects AESV2, PDF standard security handler revision 4.
	ProtectionAES128
)

var errProtectRandom = errors.New("pdf protection random source failed")

type protectType struct {
	encrypted     bool
	algorithm     ProtectionAlgorithm
	uValue        []byte
	oValue        []byte
	pValue        int
	padding       []byte
	encryptionKey []byte
	objNum        int
	fileID        []byte
	random        io.Reader
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

func (p *protectType) aesObjectKey(n uint32) []byte {
	nbuf := make([]byte, 8)
	binary.LittleEndian.PutUint32(nbuf, n)
	b := make([]byte, 0, len(p.encryptionKey)+9)
	b = append(b, p.encryptionKey...)
	b = append(b, nbuf[0], nbuf[1], nbuf[2], 0, 0)
	b = append(b, 's', 'A', 'l', 'T')
	s := md5.Sum(b) // #nosec G401 -- required by the PDF security handler.
	keyLen := min(len(p.encryptionKey)+5, aes.BlockSize)
	return s[0:keyLen]
}

func (p *protectType) encryptBytes(n uint32, data []byte) ([]byte, error) {
	if p.algorithm == ProtectionAES128 {
		return p.aesEncrypt(n, data)
	}

	buf := append([]byte(nil), data...)
	p.rc4(n, &buf)
	return buf, nil
}

func (p *protectType) aesEncrypt(n uint32, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(p.aesObjectKey(n))
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	padLen := aes.BlockSize - len(data)%aes.BlockSize
	if padLen == 0 {
		padLen = aes.BlockSize
	}

	out := make([]byte, aes.BlockSize+len(data)+padLen)
	iv := out[:aes.BlockSize]
	err = p.readRandom(iv)
	if err != nil {
		return nil, err
	}

	encrypted := out[aes.BlockSize:]
	copy(encrypted, data)
	for i := len(data); i < len(encrypted); i++ {
		encrypted[i] = byte(padLen)
	}
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, encrypted)

	return out, nil
}

func oValueGen(userPass, ownerPass []byte) []byte {
	var c *rc4.Cipher
	tmp := md5.Sum(ownerPass)      // #nosec G401 -- required by the PDF security handler.
	c, _ = rc4.NewCipher(tmp[0:5]) // #nosec G405 -- required by the PDF security handler.
	size := len(userPass)
	v := make([]byte, size)
	c.XORKeyStream(v, userPass)
	return v
}

func oValueGenRevision3(userPass, ownerPass []byte, keyLen int) []byte {
	sum := md5.Sum(ownerPass) // #nosec G401 -- required by the PDF security handler.
	digest := sum[:]
	for range 50 {
		next := md5.Sum(digest) // #nosec G401 -- required by the PDF security handler.
		digest = next[:]
	}

	key := digest[:keyLen]
	v := append([]byte(nil), userPass...)
	rc4Crypt(v, key)
	for i := 1; i <= 19; i++ {
		rc4Crypt(v, xorKey(key, byte(i)))
	}

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

func (p *protectType) uValueGenRevision3() []byte {
	buf := make([]byte, 0, len(p.padding)+len(p.fileID))
	buf = append(buf, p.padding...)
	buf = append(buf, p.fileID...)
	sum := md5.Sum(buf) // #nosec G401 -- required by the PDF security handler.

	v := append([]byte(nil), sum[:]...)
	rc4Crypt(v, p.encryptionKey)
	for i := 1; i <= 19; i++ {
		rc4Crypt(v, xorKey(p.encryptionKey, byte(i)))
	}

	uValue := make([]byte, 32)
	copy(uValue, v)
	return uValue
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
	if p.algorithm == ProtectionAES128 {
		p.setProtectionAES128(privFlag, userPassStr, ownerPassStr)
		return
	}
	p.setProtectionRC4(privFlag, userPassStr, ownerPassStr)
}

func (p *protectType) setProtectionRC4(privFlag byte, userPassStr, ownerPassStr string) {
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
		err := p.readRandom(ownerPass)
		if err != nil {
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

func (p *protectType) setProtectionAES128(privFlag byte, userPassStr, ownerPassStr string) {
	const keyLen = 16

	privFlag = 192 | (privFlag & (CnProtectCopy | CnProtectModify | CnProtectPrint | CnProtectAnnotForms))
	p.padding = []byte{
		0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
		0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
		0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
		0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
	}
	userPass := paddedProtectionPassword([]byte(userPassStr), p.padding)
	ownerPass := []byte(ownerPassStr)
	if ownerPassStr == "" {
		ownerPass = make([]byte, 8)
		err := p.readRandom(ownerPass)
		if err != nil {
			copy(ownerPass, p.padding[:8])
		}
	}
	ownerPass = paddedProtectionPassword(ownerPass, p.padding)

	p.encrypted = true
	p.fileID = make([]byte, 16)
	err := p.readRandom(p.fileID)
	if err != nil {
		copy(p.fileID, p.padding[:16])
	}
	p.oValue = oValueGenRevision3(userPass, ownerPass, keyLen)
	p.pValue = -(int(privFlag^255) + 1)
	p.encryptionKey = encryptionKeyRevision3(userPass, p.oValue, privFlag, p.fileID, keyLen)
	p.uValue = p.uValueGenRevision3()
}

func encryptionKeyRevision3(userPass, ownerValue []byte, privFlag byte, fileID []byte, keyLen int) []byte {
	buf := make([]byte, 0, len(userPass)+len(ownerValue)+4+len(fileID))
	buf = append(buf, userPass...)
	buf = append(buf, ownerValue...)
	buf = append(buf, privFlag, 0xff, 0xff, 0xff)
	buf = append(buf, fileID...)

	sum := md5.Sum(buf) // #nosec G401 -- required by the PDF security handler.
	digest := sum[:]
	for range 50 {
		next := md5.Sum(digest[:keyLen]) // #nosec G401 -- required by the PDF security handler.
		digest = next[:]
	}

	return append([]byte(nil), digest[:keyLen]...)
}

func rc4Crypt(data, key []byte) {
	c, _ := rc4.NewCipher(key) // #nosec G405 -- required by the PDF security handler.
	c.XORKeyStream(data, data)
}

func xorKey(key []byte, x byte) []byte {
	out := make([]byte, len(key))
	for i, b := range key {
		out[i] = b ^ x
	}
	return out
}

func (p *protectType) readRandom(buf []byte) error {
	reader := p.random
	if reader == nil {
		reader = cryptoRand.Reader
	}
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return errProtectRandom
	}
	return nil
}
