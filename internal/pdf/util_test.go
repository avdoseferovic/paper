package pdf

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSliceUncompressInvalidDataReturnsError(t *testing.T) {
	t.Parallel()

	out, err := sliceUncompress([]byte("not zlib data"))
	if err == nil {
		t.Fatal("expected invalid zlib data to return an error")
	}
	if out != nil {
		t.Fatalf("expected no output for invalid zlib data, got %q", out)
	}
}

func TestRemoveIntPreservesSliceWhenValueIsMissing(t *testing.T) {
	t.Parallel()

	input := []int{10, 20, 30}
	got := removeInt(input, 99)

	if !reflect.DeepEqual(got, input) {
		t.Fatalf("expected missing value removal to preserve slice; got %v", got)
	}
}

func TestUTF8ToUTF16SupplementaryPlaneUsesSurrogatePair(t *testing.T) {
	t.Parallel()

	got := []byte(utf8toutf16("😀", false))
	want := []byte{0xD8, 0x3D, 0xDE, 0x00}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestAddUTF8FontFromBytesRecordsParseErrorWithoutStdout(t *testing.T) {
	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})

	stdout := captureStdout(t, func() {
		f.AddUTF8FontFromBytes("broken", "", []byte{0, 0, 0, 0})
	})

	if stdout != "" {
		t.Fatalf("expected no stdout while parsing a bad font, got %q", stdout)
	}
	if f.Error() == nil {
		t.Fatal("expected bad font parse to be recorded on PDF")
	}
	if !strings.Contains(f.Error().Error(), "not a TrueType font") {
		t.Fatalf("expected TrueType parse error, got %q", f.Error())
	}
}

func TestAddUTF8FontFromBytesTruncatedTrueTypeDoesNotPanicOrWriteStdout(t *testing.T) {
	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})

	var recovered any
	stdout := captureStdout(t, func() {
		defer func() {
			recovered = recover()
		}()
		f.AddUTF8FontFromBytes("broken", "", []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x01})
	})

	if recovered != nil {
		t.Fatalf("expected truncated TrueType-like bytes not to panic, got %v", recovered)
	}
	if stdout != "" {
		t.Fatalf("expected no stdout while parsing a bad font, got %q", stdout)
	}
	if f.Error() == nil {
		t.Fatal("expected bad font parse to be recorded on PDF")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestRoundHandlesNegativeAndHalfValues(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{
		{0, 0},
		{0.5, 1},
		{0.49, 0},
		{1.5, 2},
		{-0.5, -1},
		{-1.5, -2},
		{-0.49, 0},
		{2.4, 2},
	}
	for _, c := range cases {
		if got := round(c.in); got != c.want {
			t.Errorf("round(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestIntIfSelectsBranch(t *testing.T) {
	if got := intIf(true, 7, 9); got != 7 {
		t.Errorf("intIf(true) = %d, want 7", got)
	}
	if got := intIf(false, 7, 9); got != 9 {
		t.Errorf("intIf(false) = %d, want 9", got)
	}
}

func TestStrIfSelectsBranch(t *testing.T) {
	if got := strIf(true, "a", "b"); got != "a" {
		t.Errorf("strIf(true) = %q, want a", got)
	}
	if got := strIf(false, "a", "b"); got != "b" {
		t.Errorf("strIf(false) = %q, want b", got)
	}
}

func TestDoNothingReturnsInput(t *testing.T) {
	if got := doNothing("unchanged"); got != "unchanged" {
		t.Errorf("doNothing = %q", got)
	}
}

func TestIsChineseDetectsCJKRange(t *testing.T) {
	if !isChinese('中') {
		t.Error("expected 中 to be detected as Chinese")
	}
	if isChinese('A') {
		t.Error("expected A not to be detected as Chinese")
	}
	if isChinese(0x4dff) {
		t.Error("expected code point just below range to be excluded")
	}
}

func TestFontFamilyEscapeReplacesSpaces(t *testing.T) {
	if got := fontFamilyEscape("Times New Roman"); got != "Times#20New#20Roman" {
		t.Errorf("fontFamilyEscape = %q", got)
	}
}

func TestRemoveIntRemovesFirstMatch(t *testing.T) {
	got := removeInt([]int{1, 2, 3, 2}, 2)
	if len(got) != 3 || got[0] != 1 || got[1] != 3 || got[2] != 2 {
		t.Errorf("removeInt = %v", got)
	}
	// Absent key returns slice unchanged.
	got = removeInt([]int{4, 5}, 9)
	if len(got) != 2 {
		t.Errorf("removeInt missing key changed slice: %v", got)
	}
}

func TestSliceCompressRoundTrip(t *testing.T) {
	original := []byte(strings.Repeat("paper pdf round trip ", 50))
	compressed := sliceCompress(original)
	if len(compressed) == 0 {
		t.Fatal("compressed output empty")
	}
	restored, err := sliceUncompress(compressed)
	if err != nil {
		t.Fatalf("sliceUncompress: %v", err)
	}
	if string(restored) != string(original) {
		t.Fatal("round-tripped data does not match original")
	}
}

func TestSliceUncompressRejectsGarbage(t *testing.T) {
	if _, err := sliceUncompress([]byte("not zlib data")); err == nil {
		t.Fatal("expected error decompressing garbage")
	}
}

func TestUtf8toutf16WithAndWithoutBOM(t *testing.T) {
	withBOM := utf8toutf16("A")
	if len(withBOM) != 4 || withBOM[0] != 0xFE || withBOM[1] != 0xFF {
		t.Fatalf("expected BOM prefix, got % x", withBOM)
	}
	noBOM := utf8toutf16("A", false)
	if len(noBOM) != 2 || noBOM[0] != 0x00 || noBOM[1] != 0x41 {
		t.Fatalf("unexpected no-BOM encoding: % x", noBOM)
	}
}

func TestUnicodeTranslatorMapsHighGlyphs(t *testing.T) {
	// One valid mapping line: glyph 0x80 -> U+20AC (euro sign).
	rep, err := UnicodeTranslator(strings.NewReader("!80 U+20AC Euro\n"))
	if err != nil {
		t.Fatalf("UnicodeTranslator: %v", err)
	}
	got := rep("€ ascii")
	// The euro should map to byte 0x80; ASCII passes through.
	if got[0] != 0x80 {
		t.Fatalf("expected euro mapped to 0x80, got % x", got)
	}
	if !strings.HasSuffix(got, " ascii") {
		t.Fatalf("ascii not preserved: %q", got)
	}
}

func TestUnicodeTranslatorUnmappedRuneBecomesDot(t *testing.T) {
	rep, err := UnicodeTranslator(strings.NewReader(""))
	if err != nil {
		t.Fatalf("UnicodeTranslator: %v", err)
	}
	// A non-ASCII rune with no mapping is replaced with '.'.
	if got := rep("é"); got != "." {
		t.Fatalf("expected unmapped rune to become '.', got %q", got)
	}
}

func TestUnicodeTranslatorMalformedLineReturnsError(t *testing.T) {
	_, err := UnicodeTranslator(strings.NewReader("this is not a valid map line\n"))
	if err == nil {
		t.Fatal("expected parse error for malformed line")
	}
}

func TestUnicodeTranslatorFromDescriptorUsesEmbeddedMap(t *testing.T) {
	f := NewCustom(&InitType{})
	rep := f.UnicodeTranslatorFromDescriptor("cp1252")
	if f.Err() {
		t.Fatalf("descriptor translator errored: %v", f.Error())
	}
	if rep == nil {
		t.Fatal("expected non-nil translator")
	}
	// Empty cpStr defaults to cp1252 and must also resolve from the embedded map.
	rep2 := f.UnicodeTranslatorFromDescriptor("")
	if rep2 == nil || f.Err() {
		t.Fatalf("default descriptor failed: %v", f.Error())
	}
}

func TestPointTypeTransformOffsets(t *testing.T) {
	p := PointType{X: 1, Y: 2}.Transform(3, 4)
	if p.X != 4 || p.Y != 6 {
		t.Fatalf("Transform = %+v", p)
	}
}

func TestSizeTypeOrientation(t *testing.T) {
	landscape := SizeType{Wd: 10, Ht: 5}
	if landscape.Orientation() != "L" {
		t.Errorf("expected L, got %q", landscape.Orientation())
	}
	portrait := SizeType{Wd: 5, Ht: 10}
	if portrait.Orientation() != "P" {
		t.Errorf("expected P, got %q", portrait.Orientation())
	}
	square := SizeType{Wd: 5, Ht: 5}
	if square.Orientation() != "" {
		t.Errorf("expected empty for square, got %q", square.Orientation())
	}
	var nilSize *SizeType
	if nilSize.Orientation() != "" {
		t.Error("expected empty for nil size")
	}
}

func TestSizeTypeScaling(t *testing.T) {
	s := SizeType{Wd: 10, Ht: 20}
	if got := s.ScaleBy(2); got.Wd != 20 || got.Ht != 40 {
		t.Errorf("ScaleBy = %+v", got)
	}
	if got := s.ScaleToWidth(5); got.Wd != 5 || got.Ht != 10 {
		t.Errorf("ScaleToWidth = %+v", got)
	}
	if got := s.ScaleToHeight(10); got.Ht != 10 || got.Wd != 5 {
		t.Errorf("ScaleToHeight = %+v", got)
	}
}
