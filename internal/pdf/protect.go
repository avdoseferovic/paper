// PDF protection is adapted from the work of Klemen VODOPIVEC for the fpdf
// product.

package pdf

import (
	"cmp"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	cryptoRand "crypto/rand"
	"crypto/rc4"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
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
}

// rc4 encrypts buf in place. The PDF standard security handler encrypts every
// string and stream independently, so a fresh cipher (and keystream) must be
// created for each call; reusing a cipher across strings of the same object
// would continue the keystream and corrupt every string after the first.
func (p *protectType) rc4(n uint32, buf *[]byte) {
	c, _ := rc4.NewCipher(p.objectKey(n))
	c.XORKeyStream(*buf, *buf)
}

func (p *protectType) objectKey(n uint32) []byte {
	nbuf := make([]byte, 8)
	binary.LittleEndian.PutUint32(nbuf, n)
	b := make([]byte, 0, len(p.encryptionKey)+5)
	b = append(b, p.encryptionKey...)
	b = append(b, nbuf[0], nbuf[1], nbuf[2], 0, 0)
	s := md5.Sum(b)
	return s[0:10]
}

func (p *protectType) aesObjectKey(n uint32) []byte {
	nbuf := make([]byte, 8)
	binary.LittleEndian.PutUint32(nbuf, n)
	b := make([]byte, 0, len(p.encryptionKey)+9)
	b = append(b, p.encryptionKey...)
	b = append(b, nbuf[0], nbuf[1], nbuf[2], 0, 0, 's', 'A', 'l', 'T')
	s := md5.Sum(b)
	keyLen := min(len(p.encryptionKey)+5, aes.BlockSize)
	return s[0:keyLen]
}

func (p *protectType) encryptBytes(n uint32, data []byte) ([]byte, error) {
	if p.algorithm == ProtectionAES128 {
		return p.aesEncrypt(n, data)
	}

	buf := slices.Clone(data)
	p.rc4(n, &buf)
	return buf, nil
}

func (p *protectType) aesEncrypt(n uint32, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(p.aesObjectKey(n))
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	// Left in explicit form: gosec proves padLen is in byte range from these
	// bounds, but cannot see through cmp.Or, and this repo keeps every integer
	// conversion provably in range rather than suppressing G115.
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
	tmp := md5.Sum(ownerPass)
	c, _ = rc4.NewCipher(tmp[0:5])
	size := len(userPass)
	v := make([]byte, size)
	c.XORKeyStream(v, userPass)
	return v
}

func oValueGenRevision3(userPass, ownerPass []byte, keyLen int) []byte {
	sum := md5.Sum(ownerPass)
	digest := sum[:]
	for range 50 {
		next := md5.Sum(digest)
		digest = next[:]
	}

	key := digest[:keyLen]
	v := slices.Clone(userPass)
	rc4Crypt(v, key)
	for i := 1; i <= 19; i++ {
		rc4Crypt(v, xorKey(key, byte(i)))
	}

	return v
}

func (p *protectType) uValueGen() []byte {
	var c *rc4.Cipher
	c, _ = rc4.NewCipher(p.encryptionKey)
	size := len(p.padding)
	v := make([]byte, size)
	c.XORKeyStream(v, p.padding)
	return v
}

func (p *protectType) uValueGenRevision3() []byte {
	buf := make([]byte, 0, len(p.padding)+len(p.fileID))
	buf = append(buf, p.padding...)
	buf = append(buf, p.fileID...)
	sum := md5.Sum(buf)

	v := slices.Clone(sum[:])
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

func (p *protectType) setProtection(privFlag byte, userPassStr, ownerPassStr string) error {
	if p.algorithm == ProtectionAES128 {
		return p.setProtectionAES128(privFlag, userPassStr, ownerPassStr)
	}
	return p.setProtectionRC4(privFlag, userPassStr, ownerPassStr)
}

// ownerPassword returns the owner password bytes, generating a random one when
// the caller supplies none. A random-source failure is reported instead of
// falling back to a constant: the padding string is published in the PDF spec,
// so a constant owner password would leave the document openly accessible while
// still looking encrypted.
func (p *protectType) ownerPassword(ownerPassStr string) ([]byte, error) {
	if ownerPassStr != "" {
		return []byte(ownerPassStr), nil
	}
	generated := make([]byte, 8)
	err := p.readRandom(generated)
	if err != nil {
		return nil, err
	}
	return generated, nil
}

func (p *protectType) setProtectionRC4(privFlag byte, userPassStr, ownerPassStr string) error {
	privFlag = 192 | (privFlag & (CnProtectCopy | CnProtectModify | CnProtectPrint | CnProtectAnnotForms))
	p.padding = []byte{
		0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
		0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
		0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
		0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
	}
	ownerPass, err := p.ownerPassword(ownerPassStr)
	if err != nil {
		return err
	}
	userPass := paddedProtectionPassword([]byte(userPassStr), p.padding)
	ownerPass = paddedProtectionPassword(ownerPass, p.padding)
	p.encrypted = true
	p.oValue = oValueGen(userPass, ownerPass)
	buf := make([]byte, 0, len(userPass)+len(p.oValue)+4)
	buf = append(buf, userPass...)
	buf = append(buf, p.oValue...)
	buf = append(buf, privFlag, 0xff, 0xff, 0xff)
	sum := md5.Sum(buf)
	p.encryptionKey = sum[0:5]
	p.uValue = p.uValueGen()
	p.pValue = -(int(privFlag^255) + 1)
	return nil
}

func (p *protectType) setProtectionAES128(privFlag byte, userPassStr, ownerPassStr string) error {
	const keyLen = 16

	privFlag = 192 | (privFlag & (CnProtectCopy | CnProtectModify | CnProtectPrint | CnProtectAnnotForms))
	p.padding = []byte{
		0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
		0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
		0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
		0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
	}
	ownerPass, err := p.ownerPassword(ownerPassStr)
	if err != nil {
		return err
	}
	userPass := paddedProtectionPassword([]byte(userPassStr), p.padding)
	ownerPass = paddedProtectionPassword(ownerPass, p.padding)

	// The file ID feeds key derivation, so a constant fallback would hand an
	// attacker the missing input for a document with no user password.
	fileID := make([]byte, 16)
	err = p.readRandom(fileID)
	if err != nil {
		return err
	}

	p.encrypted = true
	p.fileID = fileID
	p.oValue = oValueGenRevision3(userPass, ownerPass, keyLen)
	p.pValue = -(int(privFlag^255) + 1)
	p.encryptionKey = encryptionKeyRevision3(userPass, p.oValue, privFlag, p.fileID, keyLen)
	p.uValue = p.uValueGenRevision3()
	return nil
}

func encryptionKeyRevision3(userPass, ownerValue []byte, privFlag byte, fileID []byte, keyLen int) []byte {
	buf := make([]byte, 0, len(userPass)+len(ownerValue)+4+len(fileID))
	buf = append(buf, userPass...)
	buf = append(buf, ownerValue...)
	buf = append(buf, privFlag, 0xff, 0xff, 0xff)
	buf = append(buf, fileID...)

	sum := md5.Sum(buf)
	digest := sum[:]
	for range 50 {
		next := md5.Sum(digest[:keyLen])
		digest = next[:]
	}

	return slices.Clone(digest[:keyLen])
}

func rc4Crypt(data, key []byte) {
	c, _ := rc4.NewCipher(key)
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
	reader := cmp.Or(p.random, cryptoRand.Reader)
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		return errProtectRandom
	}
	return nil
}
