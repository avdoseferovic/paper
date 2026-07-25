package pdf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf16"
)

// zlibWriterPool reuses zlib (deflate) writers across sliceCompress calls.
// A fresh zlib.NewWriterLevel allocates a ~600KB flate compressor (deflate
// tables, hash chains) on first write; sliceCompress is invoked once per page
// content stream, per embedded image color/alpha plane, and per font stream,
// so a single document triggers many such allocations. zlib.Writer.Reset
// retains the underlying compressor and only re-points the output, turning the
// per-call allocation into a one-time cost amortised over the pool.
// Profiled impact: flate.NewWriter was 54% of bytes allocated via this path.
var zlibWriterPool = sync.Pool{
	New: func() any {
		w, _ := zlib.NewWriterLevel(io.Discard, zlib.BestSpeed)
		return w
	},
}

func round(f float64) int {
	if f < 0 {
		return -int(math.Floor(-f + 0.5))
	}
	return int(math.Floor(f + 0.5))
}

func sprintf(fmtStr string, args ...any) string {
	return fmt.Sprintf(fmtStr, args...)
}

// bufferFromReader returns a new buffer populated with the contents of the specified Reader
func bufferFromReader(r io.Reader) (*bytes.Buffer, error) {
	b := new(bytes.Buffer)
	_, err := b.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("read buffer: %w", err)
	}
	return b, nil
}

// sliceCompress returns a zlib-compressed copy of the specified byte array
func sliceCompress(data []byte) []byte {
	var buf bytes.Buffer
	pooled := zlibWriterPool.Get()
	cmp, ok := pooled.(*zlib.Writer)
	if !ok {
		cmp = zlib.NewWriter(&buf)
	} else {
		cmp.Reset(&buf)
	}
	_, _ = cmp.Write(data)
	_ = cmp.Close()
	zlibWriterPool.Put(cmp)
	return buf.Bytes()
}

// maxDeflateExpansion is deflate's theoretical maximum expansion ratio (1032:1)
// with slack. Any stream that claims to decode to more than this times its
// compressed size is malformed, so the ratio bounds how much memory untrusted
// compressed data can ask for.
const maxDeflateExpansion = 1100

// sliceUncompress returns an uncompressed copy of the specified zlib-compressed
// byte array, reading at most limit bytes. A limit of zero or less reads to the
// end of the stream and must only be used on data this library produced itself:
// zlib packs ~1000:1, so an unbounded read turns a small hostile stream into
// gigabytes of allocation.
func sliceUncompress(data []byte, limit int64) ([]byte, error) {
	inBuf := bytes.NewReader(data)
	r, err := zlib.NewReader(inBuf)
	if err != nil {
		return nil, fmt.Errorf("open zlib reader: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()

	var source io.Reader = r
	if limit > 0 {
		source = io.LimitReader(r, limit)
	}
	var outBuf bytes.Buffer
	_, err = outBuf.ReadFrom(source)
	if err != nil {
		return nil, fmt.Errorf("read zlib data: %w", err)
	}
	return outBuf.Bytes(), nil
}

func appendUTF16BEUnit(res []byte, unit uint16) []byte {
	return append(res, byte(unit>>8), byte(unit&0xFF))
}

// utf8toutf16 converts UTF-8 to UTF-16BE; from http://www.fpdf.org/
func utf8toutf16(s string, withBOM ...bool) string {
	bom := true
	if len(withBOM) > 0 {
		bom = withBOM[0]
	}
	res := make([]byte, 0, 8)
	if bom {
		res = append(res, 0xFE, 0xFF)
	}
	for _, unit := range utf16.Encode([]rune(s)) {
		res = appendUTF16BEUnit(res, unit)
	}
	return string(res)
}

// intIf returns a if cnd is true, otherwise b
func intIf(cnd bool, a, b int) int {
	if cnd {
		return a
	}
	return b
}

// strIf returns aStr if cnd is true, otherwise bStr
func strIf(cnd bool, aStr, bStr string) string {
	if cnd {
		return aStr
	}
	return bStr
}

// doNothing returns the passed string with no translation.
func doNothing(s string) string {
	return s
}

// Dump the internals of the specified values
// func dump(fileStr string, a ...interface{}) {
// 	fl, err := os.OpenFile(fileStr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
// 	if err == nil {
// 		fmt.Fprintf(fl, "----------------\n")
// 		spew.Fdump(fl, a...)
// 		fl.Close()
// 	}
// }

// isASCIIOnly reports whether str consists solely of bytes below 0x80.
func isASCIIOnly(str string) bool {
	for i := range len(str) {
		if str[i] >= 0x80 {
			return false
		}
	}
	return true
}

func repClosure(m map[rune]byte) func(string) string {
	var buf bytes.Buffer
	return func(str string) string {
		// Every rune below 0x80 translates to itself, so an all-ASCII string is
		// returned unchanged. Returning the input directly skips building an
		// identical copy; the translator runs on every drawn text run, and this
		// copy was ~17% of allocated objects for Latin text.
		if isASCIIOnly(str) {
			return str
		}
		return translateNonASCII(&buf, m, str)
	}
}

// translateNonASCII maps str into the code page described by m, replacing runes
// with no mapping by '.'.
func translateNonASCII(buf *bytes.Buffer, m map[rune]byte, str string) string {
	var ch byte
	var ok bool
	buf.Reset()
	for _, r := range str {
		if r < 0x80 {
			// Masking keeps the conversion provably within a byte.
			ch = byte(r & 0x7F)
		} else {
			ch, ok = m[r]
			if !ok {
				ch = byte('.')
			}
		}
		buf.WriteByte(ch)
	}
	return buf.String()
}

// UnicodeTranslator returns a function that can be used to translate, where
// possible, utf-8 strings to a form that is compatible with the specified code
// page. The returned function accepts a string and returns a string.
//
// r is a reader that should read a buffer made up of content lines that
// pertain to the code page of interest. Each line is made up of three
// whitespace separated fields. The first begins with "!" and is followed by
// two hexadecimal digits that identify the glyph position in the code page of
// interest. The second field begins with "U+" and is followed by the unicode
// code point value. The third is the glyph name. A number of these code page
// map files are packaged with the gfpdf library in the font directory.
//
// An error occurs only if a line is read that does not conform to the expected
// format. In this case, the returned function is valid but does not perform
// any rune translation.
func UnicodeTranslator(r io.Reader) (func(string) string, error) {
	m, err := parseCodepageMap(r)
	if err != nil {
		return doNothing, err
	}
	return repClosure(m), nil
}

func parseCodepageMap(r io.Reader) (map[rune]byte, error) {
	m := make(map[rune]byte)
	var uPos, cPos uint32
	var lineStr, nameStr string
	var parseErr error
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		lineStr = sc.Text()
		lineStr = strings.TrimSpace(lineStr)
		if lineStr != "" {
			_, err := fmt.Sscanf(lineStr, "!%2X U+%4X %s", &cPos, &uPos, &nameStr)
			if err == nil {
				if cPos >= 0x80 {
					m[rune(uPos)] = byte(cPos)
				}
			} else if parseErr == nil {
				parseErr = err
			}
		}
	}
	err := sc.Err()
	if err != nil {
		return nil, fmt.Errorf("scan unicode translator: %w", err)
	}
	if parseErr != nil {
		return nil, parseErr
	}
	return m, nil
}

// UnicodeTranslatorFromFile returns a function that can be used to translate,
// where possible, utf-8 strings to a form that is compatible with the
// specified code page. See UnicodeTranslator for more details.
//
// fileStr identifies a font descriptor file that maps glyph positions to names.
//
// If an error occurs reading the file, the returned function is valid but does
// not perform any rune translation.
func UnicodeTranslatorFromFile(fileStr string) (func(string) string, error) {
	fl, err := os.Open(fileStr)
	if err != nil {
		return doNothing, fmt.Errorf("open unicode translator file: %w", err)
	}
	translator, err := UnicodeTranslator(fl)
	closeErr := fl.Close()
	if err != nil {
		return translator, err
	}
	if closeErr != nil {
		return translator, fmt.Errorf("close unicode translator file: %w", closeErr)
	}
	return translator, nil
}

// codepageMapCache memoizes parsed embedded code page maps by name. Building a
// map scans ~256 descriptor lines with fmt.Sscanf, and
// UnicodeTranslatorFromDescriptor is called once per text draw, so without this
// the same static map is re-parsed thousands of times per document (profiled at
// ~30% of allocated objects). The cached maps are read-only after construction,
// so sharing them across goroutines is safe; the closure repClosure returns is
// NOT (it captures a bytes.Buffer), so a fresh closure is built per call.
var codepageMapCache sync.Map

// UnicodeTranslatorFromDescriptor returns a function that can be used to
// translate, where possible, utf-8 strings to a form that is compatible with
// the specified code page. See UnicodeTranslator for more details.
//
// cpStr identifies a code page. A descriptor file in the font directory, set
// with the fontDirStr argument in the call to New(), should have this name
// plus the extension ".map". If cpStr is empty, it will be replaced with
// "cp1252", the gofpdf code page default.
//
// If an error occurs reading the descriptor, the returned function is valid
// but does not perform any rune translation.
//
// The CellFormat_codepage example demonstrates this method.
func (f *PDF) UnicodeTranslatorFromDescriptor(cpStr string) func(string) string {
	if f.err != nil {
		return doNothing
	}
	if cpStr == "" {
		cpStr = "cp1252"
	}

	if cached, hit := codepageMapCache.Load(cpStr); hit {
		if m, valid := cached.(map[rune]byte); valid {
			return repClosure(m)
		}
	}

	// Code pages Paper ships are parsed from the embedded descriptor and cached;
	// anything else is read from the font directory.
	str, ok := embeddedCodepageMap(cpStr)
	if !ok {
		rep, err := UnicodeTranslatorFromFile(filepath.Join(f.fontpath, cpStr) + ".map")
		f.SetError(err)
		return rep
	}

	m, err := parseCodepageMap(strings.NewReader(str))
	f.SetError(err)
	if err != nil {
		return doNothing
	}
	codepageMapCache.Store(cpStr, m)
	return repClosure(m)
}

// Transform moves a point by given X, Y offset
func (p PointType) Transform(x, y float64) PointType {
	return PointType{p.X + x, p.Y + y}
}

// Orientation returns the orientation of a given size:
// "P" for portrait, "L" for landscape
func (s *SizeType) Orientation() string {
	if s == nil || s.Ht == s.Wd {
		return ""
	}
	if s.Wd > s.Ht {
		return "L"
	}
	return "P"
}

// ScaleBy expands a size by a certain factor
func (s *SizeType) ScaleBy(factor float64) SizeType {
	return SizeType{s.Wd * factor, s.Ht * factor}
}

// ScaleToWidth adjusts the height of a size to match the given width
func (s *SizeType) ScaleToWidth(width float64) SizeType {
	height := s.Ht * width / s.Wd
	return SizeType{width, height}
}

// ScaleToHeight adjusts the width of a size to match the given height
func (s *SizeType) ScaleToHeight(height float64) SizeType {
	width := s.Wd * height / s.Ht
	return SizeType{width, height}
}

func isChinese(rune2 rune) bool {
	// chinese unicode: 4e00-9fa5
	if rune2 >= rune(0x4e00) && rune2 <= rune(0x9fa5) {
		return true
	}
	return false
}

// Condition font family string to PDF name compliance. See section 5.3 (Names)
// in https://resources.infosecinstitute.com/pdf-file-format-basic-structure/
func fontFamilyEscape(familyStr string) string {
	return pdfNameEscape(familyStr)
}

// pdfNameByteNeedsEscape reports whether c must be written as #XX inside a PDF
// name object: every byte outside the regular printable range, and every
// delimiter or '#'.
func pdfNameByteNeedsEscape(c byte) bool {
	if c < '!' || c > '~' {
		return true
	}
	switch c {
	case '#', '/', '%', '(', ')', '<', '>', '[', ']', '{', '}':
		return true
	}
	return false
}

const pdfNameHexDigits = "0123456789ABCDEF"

// pdfNameEscape escapes a string for use as a PDF name object (PDF 32000-1
// §7.3.5): every byte outside the regular printable range and every
// delimiter or '#' is written as #XX.
//
// Names that need no escaping — the common case, since this is on the path of
// every SetFont call — are returned unchanged so the caller allocates nothing.
func pdfNameEscape(s string) string {
	needsEscape := false
	for i := range len(s) {
		if pdfNameByteNeedsEscape(s[i]) {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := range len(s) {
		c := s[i]
		if pdfNameByteNeedsEscape(c) {
			b.WriteByte('#')
			b.WriteByte(pdfNameHexDigits[c>>4])
			b.WriteByte(pdfNameHexDigits[c&0x0F])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
