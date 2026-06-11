package pdf

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOkClearErrorAndString(t *testing.T) {
	f := readyPDF(t)
	if !f.Ok() {
		t.Fatalf("expected fresh document to be ok, got %v", f.Error())
	}
	if got := f.String(); !strings.HasPrefix(got, "PDF ") {
		t.Fatalf("expected String() to summarize the instance, got %q", got)
	}

	f.SetErrorf("synthetic failure")
	if f.Ok() {
		t.Fatal("expected Ok() to be false after an error is set")
	}
	f.ClearError()
	if !f.Ok() {
		t.Fatal("expected ClearError to reset the error state")
	}
}

func TestSetDisplayModeLayoutsRender(t *testing.T) {
	f := readyPDF(t)
	f.SetCompression(false)
	f.SetDisplayMode("fullwidth", "TwoColumnLeft")
	if f.Err() {
		t.Fatalf("SetDisplayMode errored: %v", f.Error())
	}
	out := string(mustOutput(t, f))
	if !strings.Contains(out, "/PageLayout /TwoColumnLeft") {
		t.Fatal("expected page layout entry in catalog")
	}

	for _, tc := range []struct {
		zoom, layout, marker string
	}{
		{"fullpage", "single", "/OpenAction [3 0 R /Fit]"},
		{"real", "continuous", "/PageLayout /OneColumn"},
		{"default", "two", "/PageLayout /TwoColumnLeft"},
		{"fullwidth", "TwoPageRight", "/PageLayout /TwoPageRight"},
	} {
		doc := readyPDF(t)
		doc.SetCompression(false)
		doc.SetDisplayMode(tc.zoom, tc.layout)
		if doc.Err() {
			t.Fatalf("SetDisplayMode(%q, %q) errored: %v", tc.zoom, tc.layout, doc.Error())
		}
		body := string(mustOutput(t, doc))
		if !strings.Contains(body, tc.marker) {
			t.Fatalf("expected %q in catalog for mode (%q, %q)", tc.marker, tc.zoom, tc.layout)
		}
	}

	bad := readyPDF(t)
	bad.SetDisplayMode("bogus-zoom", "single")
	if !bad.Err() {
		t.Fatal("expected error for invalid zoom mode")
	}

	badLayout := readyPDF(t)
	badLayout.SetDisplayMode("fullpage", "bogus-layout")
	if !badLayout.Err() {
		t.Fatal("expected error for invalid layout mode")
	}
}

func TestTextCoreFontUnderlineStrikeoutAndColor(t *testing.T) {
	f := readyPDF(t)
	f.SetCompression(false)
	f.SetFont("Helvetica", "US", 12)
	f.SetTextColor(200, 0, 0)
	f.Text(20, 30, "decorated")
	if f.Err() {
		t.Fatalf("Text errored: %v", f.Error())
	}

	content := f.pages[f.page].String()
	if !strings.Contains(content, "(decorated) Tj") {
		t.Fatal("expected escaped core-font text operator")
	}
	if strings.Count(content, "re f") < 2 {
		t.Fatal("expected underline and strikeout rectangles")
	}
	if !strings.Contains(content, "q ") || !strings.Contains(content, " Q") {
		t.Fatal("expected text color to wrap the text operation")
	}
}

func TestTextUTF8RTLReversesString(t *testing.T) {
	f := readyUTF8PDF(t)
	f.RTL()
	f.Text(100, 30, "abc")
	if f.Err() {
		t.Fatalf("Text errored: %v", f.Error())
	}
	if !strings.Contains(f.pages[f.page].String(), "Tj ET") {
		t.Fatal("expected RTL text operator in page content")
	}
}

func TestHeaderFuncModeAndFooterFuncLpi(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetFont("Helvetica", "", 12)

	headerCalls := 0
	footerCalls := 0
	sawLastPage := false
	f.SetHeaderFuncMode(func() {
		headerCalls++
		f.Cell(40, 10, "header")
	}, true)
	f.SetFooterFuncLpi(func(lastPage bool) {
		footerCalls++
		if lastPage {
			sawLastPage = true
		}
	})

	f.AddPage()
	f.AddPage()
	mustOutput(t, f)

	if headerCalls != 2 {
		t.Fatalf("expected header on each page, got %d calls", headerCalls)
	}
	if footerCalls < 2 {
		t.Fatalf("expected footer on each page, got %d calls", footerCalls)
	}
	if !sawLastPage {
		t.Fatal("expected footer function to receive the last-page indicator")
	}
}

func TestUnicodeTranslatorFromFile(t *testing.T) {
	tr, err := UnicodeTranslatorFromFile(filepath.Join("embedded", "maps", "cp1252.map"))
	if err != nil {
		t.Fatalf("expected map file to load, got %v", err)
	}
	if got := tr("abc"); got != "abc" {
		t.Fatalf("expected ASCII passthrough, got %q", got)
	}

	tr, err = UnicodeTranslatorFromFile(filepath.Join(t.TempDir(), "missing.map"))
	if err == nil {
		t.Fatal("expected error for missing map file")
	}
	if tr == nil || tr("abc") != "abc" {
		t.Fatal("expected valid no-op translator on error")
	}
}

func TestSetAlphaBlendModeWrittenToOutput(t *testing.T) {
	f := readyPDF(t)
	f.SetCompression(false)
	f.SetAlpha(0.5, "Multiply")
	f.Rect(10, 10, 30, 30, "F")
	out := string(mustOutput(t, f))
	if !strings.Contains(out, "/BM /Multiply") {
		t.Fatal("expected blend mode ExtGState object in output")
	}
}

func TestSetDashPatternWritesBufferedOps(t *testing.T) {
	f := readyPDF(t)
	f.SetCompression(false)
	f.SetDashPattern([]float64{3, 1}, 0.5)
	f.Line(10, 10, 50, 10)
	want := sprintf("[%.2f %.2f] %.2f d", 3*f.k, 1*f.k, 0.5*f.k)
	out := string(mustOutput(t, f))
	if !strings.Contains(out, want) {
		t.Fatalf("expected dash pattern operator %q in page stream", want)
	}
}

func TestRegisterImageFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "img.png")
	if err := os.WriteFile(path, pngImageBytes(t), 0o644); err != nil {
		t.Fatalf("write PNG fixture: %v", err)
	}

	f := readyPDF(t)
	info := f.RegisterImage(path, "")
	if info == nil || f.Err() {
		t.Fatalf("expected image registration to succeed, err=%v", f.Error())
	}
	if again := f.RegisterImage(path, ""); again != info {
		t.Fatal("expected repeated registration to return the cached image info")
	}
	f.Image(path, 10, 10, 20, 20, false, "", 0, "")
	mustOutput(t, f)

	missing := readyPDF(t)
	if got := missing.RegisterImage(filepath.Join(dir, "missing.png"), ""); got != nil || !missing.Err() {
		t.Fatal("expected error for missing image file")
	}
}

func TestRegisterImageOptionsUntypedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "imagewithoutextension")
	if err := os.WriteFile(path, pngImageBytes(t), 0o644); err != nil {
		t.Fatalf("write PNG fixture: %v", err)
	}

	f := readyPDF(t)
	f.RegisterImageOptions(path, ImageOptions{})
	if !f.Err() {
		t.Fatal("expected error for image file without type information")
	}
}

// writePNGChunk appends one PNG chunk; the parser skips CRC validation, so a
// zero CRC is sufficient.
func writePNGChunk(buf *bytes.Buffer, chunkType string, data []byte) {
	buf.Write(appendUint32(nil, len(data)))
	buf.WriteString(chunkType)
	buf.Write(data)
	buf.Write([]byte{0, 0, 0, 0})
}

// grayAlphaPNGBytes hand-crafts a 2x2 grayscale+alpha (color type 4) PNG,
// which Go's encoder never emits.
func grayAlphaPNGBytes(t *testing.T) []byte {
	t.Helper()
	const w, h = 2, 2

	var raw bytes.Buffer
	for y := 0; y < h; y++ {
		raw.WriteByte(0) // filter: none
		for x := 0; x < w; x++ {
			raw.WriteByte(byte(50 + x*10 + y*20)) // gray
			raw.WriteByte(byte(100 + x*5 + y*3))  // alpha
		}
	}
	var idat bytes.Buffer
	zw := zlib.NewWriter(&idat)
	if _, err := zw.Write(raw.Bytes()); err != nil {
		t.Fatalf("compress PNG scanlines: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zlib writer: %v", err)
	}

	var png bytes.Buffer
	png.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := appendUint32(nil, w)
	ihdr = appendUint32(ihdr, h)
	ihdr = append(ihdr, 8, 4, 0, 0, 0) // bit depth 8, color type 4 (gray+alpha)
	writePNGChunk(&png, "IHDR", ihdr)
	writePNGChunk(&png, "IDAT", idat.Bytes())
	writePNGChunk(&png, "IEND", nil)
	return png.Bytes()
}

func TestRegisterGrayAlphaPNGSplitsSoftMask(t *testing.T) {
	f := readyPDF(t)
	info := f.RegisterImageOptionsReader("gray-alpha", ImageOptions{ImageType: "png"}, bytes.NewReader(grayAlphaPNGBytes(t)))
	if f.Err() {
		t.Fatalf("registering gray+alpha PNG errored: %v", f.Error())
	}
	if info == nil {
		t.Fatal("expected image info for gray+alpha PNG")
	}
	if len(info.smask) == 0 {
		t.Fatal("expected alpha channel to be split into a soft mask")
	}
	if info.cs != colorSpaceDeviceGray {
		t.Fatalf("expected DeviceGray color space, got %q", info.cs)
	}
	f.Image("gray-alpha", 10, 10, 20, 20, false, "", 0, "")
	mustOutput(t, f)
}

func TestPNGTransparencyChunkParsing(t *testing.T) {
	f := readyPDF(t)

	if got := f.pngTransparency(0, []byte{0, 7}); len(got) != 1 || got[0] != 7 {
		t.Fatalf("unexpected grayscale transparency: %v", got)
	}
	if got := f.pngTransparency(2, []byte{0, 1, 0, 2, 0, 3}); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("unexpected RGB transparency: %v", got)
	}
	if got := f.pngTransparency(3, []byte{0xFF, 0xFF, 0x00, 0xFF}); len(got) != 1 || got[0] != 2 {
		t.Fatalf("unexpected indexed transparency: %v", got)
	}
	if got := f.pngTransparency(3, []byte{0xFF, 0xFF}); got != nil {
		t.Fatalf("expected nil for indexed chunk without transparent entry, got %v", got)
	}
	if f.Err() {
		t.Fatalf("unexpected error: %v", f.Error())
	}

	f.pngTransparency(0, []byte{0})
	if !f.Err() {
		t.Fatal("expected error for truncated grayscale transparency chunk")
	}
	f.ClearError()
	f.pngTransparency(2, []byte{0, 1})
	if !f.Err() {
		t.Fatal("expected error for truncated RGB transparency chunk")
	}
}

func TestApplyPNGPhysicalDimensions(t *testing.T) {
	f := readyPDF(t)

	meterChunk := appendUint32(nil, 11811) // 300 dpi in pixels per meter
	meterChunk = appendUint32(meterChunk, 11811)
	meterChunk = append(meterChunk, 1)
	info := f.newImageInfo()
	f.applyPNGPhysicalDimensions(meterChunk, true, info)
	if info.dpi < 299 || info.dpi > 301 {
		t.Fatalf("expected ~300 dpi from meter units, got %v", info.dpi)
	}

	rawChunk := appendUint32(nil, 144)
	rawChunk = appendUint32(rawChunk, 144)
	rawChunk = append(rawChunk, 0)
	info = f.newImageInfo()
	f.applyPNGPhysicalDimensions(rawChunk, true, info)
	if info.dpi != 144 {
		t.Fatalf("expected raw dpi 144, got %v", info.dpi)
	}

	// Mismatched x/y density or readDPI disabled leaves the default.
	mismatch := appendUint32(nil, 100)
	mismatch = appendUint32(mismatch, 200)
	mismatch = append(mismatch, 1)
	info = f.newImageInfo()
	f.applyPNGPhysicalDimensions(mismatch, true, info)
	if info.dpi != 72 {
		t.Fatalf("expected default dpi for mismatched density, got %v", info.dpi)
	}

	f.applyPNGPhysicalDimensions([]byte{1, 2, 3}, true, info)
	if !f.Err() {
		t.Fatal("expected error for truncated pHYs chunk")
	}
}
