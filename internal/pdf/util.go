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

// sliceUncompress returns an uncompressed copy of the specified zlib-compressed byte array
func sliceUncompress(data []byte) ([]byte, error) {
	inBuf := bytes.NewReader(data)
	r, err := zlib.NewReader(inBuf)
	if err != nil {
		return nil, fmt.Errorf("open zlib reader: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()

	var outBuf bytes.Buffer
	_, err = outBuf.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("read zlib data: %w", err)
	}
	return outBuf.Bytes(), nil
}

func appendUTF16BEUnit(res []byte, unit uint16) []byte {
	return append(res, byte(unit>>8), byte(unit)) // #nosec G115 -- bytes are the high and low halves of a uint16.
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

func repClosure(m map[rune]byte) func(string) string {
	var buf bytes.Buffer
	return func(str string) string {
		var ch byte
		var ok bool
		buf.Reset()
		for _, r := range str {
			if r < 0x80 {
				ch = byte(r) // #nosec G115 -- branch guarantees an ASCII byte.
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
	m := make(map[rune]byte)
	var uPos, cPos uint32
	var lineStr, nameStr string
	var parseErr error
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		lineStr = sc.Text()
		lineStr = strings.TrimSpace(lineStr)
		if len(lineStr) > 0 {
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
		return doNothing, fmt.Errorf("scan unicode translator: %w", err)
	}
	if parseErr != nil {
		return doNothing, parseErr
	}
	return repClosure(m), nil
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
	var str string
	var ok bool
	if f.err == nil {
		if len(cpStr) == 0 {
			cpStr = "cp1252"
		}
		str, ok = embeddedMapList[cpStr]
		if ok {
			var err error
			rep, err := UnicodeTranslator(strings.NewReader(str))
			f.SetError(err)
			return rep
		} else {
			var err error
			rep, err := UnicodeTranslatorFromFile(filepath.Join(f.fontpath, cpStr) + ".map")
			f.SetError(err)
			return rep
		}
	}
	return doNothing
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

func removeInt(arr []int, key int) []int {
	for i, mKey := range arr {
		if mKey == key {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
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
	escStr := strings.ReplaceAll(familyStr, " ", "#20")
	// Additional replacements can take place here
	return escStr
}
