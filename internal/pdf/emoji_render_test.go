package pdf

import (
	"strings"
	"testing"
)

// colorEmojiPDF returns a UTF-8 PDF whose current font reports COLR/CPAL color
// glyph data for the rune 'A'. The layers reference the real Arial Unicode
// outline for that glyph, so rendering produces actual path operators.
func colorEmojiPDF(t *testing.T) *PDF {
	t.Helper()
	f := readyUTF8PDF(t)
	utf := f.currentFont.utf8File
	gid, ok := utf.charSymbolDictionary[int('A')]
	if !ok {
		t.Fatal("expected glyph mapping for 'A' in test font")
	}
	gid16, ok := checkedUint16(gid)
	if !ok {
		t.Fatalf("glyph id out of range: %d", gid)
	}
	utf.colrTable = &colrTable{
		baseGlyphRecords: []colorBaseGlyphRecord{
			{glyphID: gid16, firstLayerIdx: 0, numLayers: 1},
		},
		layerRecords: []colorLayerRecord{
			{glyphID: gid16, paletteIndex: 0},
		},
	}
	utf.cpalTable = &cpalTable{colorRecords: []colorRecord{{r: 200, g: 10, b: 30, a: 255}}}
	utf.hasColorGlyphs = true
	f.currentFont.hasColorGlyphs = true
	f.SetColorEmojiEnabled(true)
	return f
}

func TestTextRendersCOLRColorGlyphLayers(t *testing.T) {
	f := colorEmojiPDF(t)
	f.SetTextColor(20, 30, 40)
	f.Text(10, 20, "AB")
	if f.Err() {
		t.Fatalf("Text errored: %v", f.Error())
	}

	content := f.pages[f.page].String()
	if !strings.Contains(content, "0.784 0.039 0.118 rg") {
		t.Fatal("expected color glyph layer fill color in page content")
	}
	if !strings.Contains(content, "f Q") {
		t.Fatal("expected filled color glyph path in page content")
	}
	if !strings.Contains(content, "3 Tr") {
		t.Fatal("expected invisible text operator behind the color glyph")
	}
	if !strings.Contains(content, "Tj ET") {
		t.Fatal("expected monochrome glyph text segment for 'B'")
	}
	mustOutput(t, f)
}

func TestTextColorEmojiRTLWithUnderlineAndStrikeout(t *testing.T) {
	f := colorEmojiPDF(t)
	f.SetFont("arial", "US", 12)
	f.SetColorEmojiEnabled(true)
	f.currentFont.hasColorGlyphs = true
	f.RTL()
	f.Text(100, 20, "AB")
	if f.Err() {
		t.Fatalf("Text errored: %v", f.Error())
	}

	content := f.pages[f.page].String()
	if !strings.Contains(content, "re f") {
		t.Fatal("expected underline/strikeout rectangles in page content")
	}
	mustOutput(t, f)
}

// bitmapEmojiFontFile builds a synthetic CBDT/CBLC font file containing one
// real PNG glyph for glyph id 7.
func bitmapEmojiFontFile(t *testing.T) *utf8FontFile {
	t.Helper()
	png := pngImageBytes(t)
	glyphData := []byte{12, 10, 0xff, 9, 11} // height, width, bearingX(-1), bearingY, advance
	glyphData = appendUint32(glyphData, len(png))
	glyphData = append(glyphData, png...)

	cblc := buildTestCBLCFormat17Table(7, 18, len(glyphData))
	cbdt := appendUint32(nil, 0x00030000)
	cbdt = append(cbdt, glyphData...)

	data := append(append([]byte(nil), cblc...), cbdt...)
	utf := &utf8FontFile{
		fileReader: &fileReader{array: data},
		tableDescriptions: map[string]*tableDescription{
			"CBLC": {position: 0, size: len(cblc)},
			"CBDT": {position: len(cblc), size: len(cbdt)},
		},
	}
	utf.parseCBLCTable()
	if utf.err != nil {
		t.Fatalf("parse synthetic CBLC table: %v", utf.err)
	}
	utf.hasBitmapGlyphs = true
	utf.hasColorGlyphs = true
	utf.charSymbolDictionary = map[int]int{0x1F600: 7}
	return utf
}

func TestTextWithColorEmojiRendersBitmapGlyph(t *testing.T) {
	f := readyUTF8PDF(t)
	f.SetColorEmojiEnabled(true)
	f.currentFont.utf8File = bitmapEmojiFontFile(t)
	f.currentFont.hasColorGlyphs = true
	f.currentFont.Tp = fontTypeUTF8Bitmap

	if !f.textContainsColorEmoji("x\U0001F600") {
		t.Fatal("expected bitmap emoji rune to be detected as a color glyph")
	}
	f.textWithColorEmoji(10, 20, "x\U0001F600")
	if f.Err() {
		t.Fatalf("textWithColorEmoji errored: %v", f.Error())
	}

	content := f.pages[f.page].String()
	if !strings.Contains(content, "Do Q") {
		t.Fatal("expected bitmap glyph image placement operator in page content")
	}
}

func TestRenderBitmapGlyphFallbackMetricsAndEmptyGlyph(t *testing.T) {
	f := readyUTF8PDF(t)

	if got := f.renderBitmapGlyph(9, &bitmapGlyphImage{}, 0, 0); got != "" {
		t.Fatalf("expected empty result for empty glyph, got %q", got)
	}

	glyph := &bitmapGlyphImage{
		data:          pngImageBytes(t),
		imageType:     "png",
		width:         4,
		height:        4,
		originOffsetX: 1,
		originOffsetY: 2,
	}
	got := f.renderBitmapGlyph(9, glyph, 10, 20)
	if !strings.Contains(got, "Do Q") {
		t.Fatalf("expected image placement operator, got %q", got)
	}
	if f.Err() {
		t.Fatalf("renderBitmapGlyph errored: %v", f.Error())
	}
}

// buildSimpleTestGlyph returns glyf data for a one-contour, three-point glyph
// using the repeat-flag encoding.
func buildSimpleTestGlyph() []byte {
	var g []byte
	g = appendUint16(g, 1)    // numContours
	g = appendInt16(g, 0)     // xMin
	g = appendInt16(g, 0)     // yMin
	g = appendInt16(g, 100)   // xMax
	g = appendInt16(g, 100)   // yMax
	g = appendUint16(g, 2)    // endPtsOfContours[0]
	g = appendUint16(g, 0)    // instructionLength
	g = append(g, 0x3F, 2)    // onCurve|xShort|yShort|repeat|xSame|ySame, repeat 2
	g = append(g, 10, 20, 30) // x deltas
	g = append(g, 5, 6, 7)    // y deltas
	return g
}

// buildCompositeTestGlyph returns a composite glyph referencing glyph 1 four
// times with different argument and scale encodings.
func buildCompositeTestGlyph() []byte {
	var g []byte
	g = appendInt16(g, -1)  // numContours: composite
	g = appendInt16(g, 0)   // xMin
	g = appendInt16(g, 0)   // yMin
	g = appendInt16(g, 200) // xMax
	g = appendInt16(g, 200) // yMax

	// Component 1: word args with a single scale.
	g = appendUint16(g, symbolWords|symbolScale|symbolContinue)
	g = appendUint16(g, 1)
	g = appendInt16(g, 100)
	g = appendInt16(g, -50)
	g = appendUint16(g, 0x4000) // 1.0 in 2.14

	// Component 2: byte args with separate x/y scales.
	g = appendUint16(g, symbolAllScale|symbolContinue)
	g = appendUint16(g, 1)
	g = append(g, 3, 0xFC) // args 3, -4
	g = appendUint16(g, 0x2000)
	g = appendUint16(g, 0x4000)

	// Component 3: byte args with a 2x2 transform.
	g = appendUint16(g, symbol2x2|symbolContinue)
	g = appendUint16(g, 1)
	g = append(g, 0, 0)
	g = appendUint16(g, 0x4000)
	g = appendUint16(g, 0)
	g = appendUint16(g, 0)
	g = appendUint16(g, 0x4000)

	// Component 4: word args with no transform, last component.
	g = appendUint16(g, symbolWords)
	g = appendUint16(g, 1)
	g = appendInt16(g, 0)
	g = appendInt16(g, 0)
	return g
}

func TestParseGlyphOutlineCompositeGlyph(t *testing.T) {
	t.Parallel()

	composite := buildCompositeTestGlyph()
	simple := buildSimpleTestGlyph()
	glyf := append(append([]byte(nil), composite...), simple...)
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: glyf},
		tableDescriptions: map[string]*tableDescription{"glyf": {position: 0, size: len(glyf)}},
		symbolPosition:    []int{0, len(composite), len(composite) + len(simple)},
	}

	outline := utf.parseGlyphOutline(0)
	if outline == nil {
		t.Fatal("expected composite glyph outline")
	}
	if len(outline.contours) != 4 {
		t.Fatalf("expected four transformed contours, got %d", len(outline.contours))
	}
	first := outline.contours[0]
	if len(first) != 3 {
		t.Fatalf("expected three points per contour, got %d", len(first))
	}
	if first[0].x != 110 || first[0].y != -45 {
		t.Fatalf("expected first point translated to (110, -45), got (%v, %v)", first[0].x, first[0].y)
	}
	scaled := outline.contours[1]
	if scaled[0].x != 8 || scaled[0].y != 1 {
		t.Fatalf("expected scaled point (8, 1), got (%v, %v)", scaled[0].x, scaled[0].y)
	}
	untransformed := outline.contours[3]
	if untransformed[0].x != 10 || untransformed[0].y != 5 {
		t.Fatalf("expected untransformed point (10, 5), got (%v, %v)", untransformed[0].x, untransformed[0].y)
	}
}

func TestGetOrAssignCIDFallsBackOutsidePrivateUseArea(t *testing.T) {
	t.Parallel()

	f := &PDF{}
	f.currentFont.runeToCID = make(map[int]int)
	f.currentFont.usedRunes = make(map[int]int)
	for cid := 0xE000; cid <= 0xF8FF; cid++ {
		f.currentFont.usedRunes[cid] = 1
	}

	cid := f.getOrAssignCID(0x1F700)
	if cid != 32 {
		t.Fatalf("expected first free CID 32 once the PUA is exhausted, got %d", cid)
	}

	for c := 32; c <= 0xFFFF; c++ {
		f.currentFont.usedRunes[c] = 1
	}
	if got := f.findNextFreeCID(); got != 0 {
		t.Fatalf("expected 0 when all CIDs are taken, got %d", got)
	}
}

func TestGetOrAssignCIDRemapsCollidingRune(t *testing.T) {
	t.Parallel()

	f := &PDF{}
	f.currentFont.runeToCID = make(map[int]int)
	f.currentFont.usedRunes = map[int]int{100: 999}

	cid := f.getOrAssignCID(100)
	if cid != 0xE000 {
		t.Fatalf("expected colliding rune to be remapped into the PUA, got %#x", cid)
	}
}
