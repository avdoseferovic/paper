package sign

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

const (
	contentsPlaceholderLen = 32768
	byteRangeWidth         = 10
)

type signaturePlaceholder struct {
	byteRangeOffset int
	contentsOffset  int
	contentsLen     int
}

var (
	byteRangePlaceholder = fmt.Sprintf("[%s %s %s %s]",
		strings.Repeat("0", byteRangeWidth),
		strings.Repeat("0", byteRangeWidth),
		strings.Repeat("0", byteRangeWidth),
		strings.Repeat("0", byteRangeWidth),
	)
	contentsPlaceholder = "<" + strings.Repeat("0", contentsPlaceholderLen) + ">"
)

var (
	errSignatureObjectNotFound = errors.New("sign: signature object not found")
	errByteRangeNotFound       = errors.New("sign: could not find /ByteRange placeholder")
	errContentsNotFound        = errors.New("sign: could not find /Contents placeholder")
	errSignatureTooLarge       = errors.New("sign: signature too large")
)

func buildSigDict(name, location, reason, contactInfo string) []byte {
	var out strings.Builder
	out.WriteString("<< /Type /Sig")
	out.WriteString(" /Filter /Adobe.PPKLite")
	out.WriteString(" /SubFilter /ETSI.CAdES.detached")
	if name != "" {
		fmt.Fprintf(&out, " /Name (%s)", escapePDFString(name))
	}
	if location != "" {
		fmt.Fprintf(&out, " /Location (%s)", escapePDFString(location))
	}
	if reason != "" {
		fmt.Fprintf(&out, " /Reason (%s)", escapePDFString(reason))
	}
	if contactInfo != "" {
		fmt.Fprintf(&out, " /ContactInfo (%s)", escapePDFString(contactInfo))
	}
	out.WriteString(" /ByteRange ")
	out.WriteString(byteRangePlaceholder)
	out.WriteString(" /Contents ")
	out.WriteString(contentsPlaceholder)
	out.WriteString(" >>")
	return []byte(out.String())
}

func locatePlaceholders(pdf []byte, sigDictObjNum int) (signaturePlaceholder, error) {
	var ph signaturePlaceholder
	objHeader := fmt.Sprintf("%d 0 obj", sigDictObjNum)
	objStart := bytes.Index(pdf, []byte(objHeader))
	if objStart < 0 {
		return ph, fmt.Errorf("%w: %d", errSignatureObjectNotFound, sigDictObjNum)
	}
	searchArea := pdf[objStart:]

	brMarker := []byte("/ByteRange ")
	brIdx := bytes.Index(searchArea, brMarker)
	if brIdx < 0 {
		return ph, errByteRangeNotFound
	}
	ph.byteRangeOffset = objStart + brIdx + len(brMarker)

	contentsMarker := []byte("/Contents <")
	cIdx := bytes.Index(searchArea, contentsMarker)
	if cIdx < 0 {
		return ph, errContentsNotFound
	}
	ph.contentsOffset = objStart + cIdx + len("/Contents ")
	ph.contentsLen = len(contentsPlaceholder)
	return ph, nil
}

func patchByteRange(pdf []byte, ph signaturePlaceholder) {
	fileLen := len(pdf)
	contentsStart := ph.contentsOffset
	contentsEnd := ph.contentsOffset + ph.contentsLen

	br := fmt.Sprintf("[%0*d %0*d %0*d %0*d]",
		byteRangeWidth, 0,
		byteRangeWidth, contentsStart,
		byteRangeWidth, contentsEnd,
		byteRangeWidth, fileLen-contentsEnd,
	)
	copy(pdf[ph.byteRangeOffset:], br)
}

func patchContents(pdf []byte, ph signaturePlaceholder, sig []byte) error {
	maxSigLen := contentsPlaceholderLen / 2
	if len(sig) > maxSigLen {
		return fmt.Errorf("%w: %d bytes, max %d", errSignatureTooLarge, len(sig), maxSigLen)
	}
	hexSig := fmt.Sprintf("%X", sig)
	hexSig += strings.Repeat("0", contentsPlaceholderLen-len(hexSig))
	copy(pdf[ph.contentsOffset+1:], hexSig)
	return nil
}

func escapePDFString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
