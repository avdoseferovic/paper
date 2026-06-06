package paperpdf

import "testing"

func TestUTF8FontFileParseCMAPPrefersFormat12(t *testing.T) {
	t.Parallel()

	cmap := buildTestCMAPTable()
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: cmap},
		tableDescriptions: map[string]*tableDescription{"cmap": {position: 0, size: len(cmap)}},
	}

	got := utf.parseCMAPTable(0)
	if got != 36 {
		t.Fatalf("expected format-12 cmap at offset 36, got %d", got)
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileGenerateSCCSDictionariesSupportsFormat12(t *testing.T) {
	t.Parallel()

	cmap := buildTestCMAPTable()
	utf := &utf8FontFile{
		fileReader: &fileReader{array: cmap},
	}
	symbolCharDictionary := make(map[int][]int)
	charSymbolDictionary := make(map[int]int)

	utf.generateSCCSDictionaries(36, symbolCharDictionary, charSymbolDictionary)

	if got := charSymbolDictionary[0x1F600]; got != 42 {
		t.Fatalf("expected U+1F600 to map to glyph 42, got %d", got)
	}
	if got := symbolCharDictionary[42]; len(got) != 1 || got[0] != 0x1F600 {
		t.Fatalf("expected glyph 42 to map back to U+1F600, got %v", got)
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileColorGlyphLayersV1(t *testing.T) {
	t.Parallel()

	colr := buildTestCOLRV1Table()
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: colr},
		tableDescriptions: map[string]*tableDescription{"COLR": {position: 0, size: len(colr)}},
	}

	utf.parseCOLRTable()
	layers := utf.colorGlyphLayers(42)

	if len(layers) != 1 {
		t.Fatalf("expected one COLRv1 layer, got %v", layers)
	}
	if layers[0].glyphID != 77 || layers[0].paletteIndex != 3 {
		t.Fatalf("expected glyph 77 with palette index 3, got %+v", layers[0])
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileColorGlyphLayersV1PaintColrGlyph(t *testing.T) {
	t.Parallel()

	colr := buildTestCOLRV1ReuseTable()
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: colr},
		tableDescriptions: map[string]*tableDescription{"COLR": {position: 0, size: len(colr)}},
	}

	utf.parseCOLRTable()
	layers := utf.colorGlyphLayers(42)

	if len(layers) != 1 {
		t.Fatalf("expected one reused COLRv1 layer, got %v", layers)
	}
	if layers[0].glyphID != 77 || layers[0].paletteIndex != 3 {
		t.Fatalf("expected glyph 77 with palette index 3, got %+v", layers[0])
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileColorGlyphLayersV1PaintComposite(t *testing.T) {
	t.Parallel()

	colr := buildTestCOLRV1CompositeTable()
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: colr},
		tableDescriptions: map[string]*tableDescription{"COLR": {position: 0, size: len(colr)}},
	}

	utf.parseCOLRTable()
	layers := utf.colorGlyphLayers(42)

	if len(layers) != 2 {
		t.Fatalf("expected two composite COLRv1 layers, got %v", layers)
	}
	if layers[0].glyphID != 88 || layers[0].paletteIndex != 4 {
		t.Fatalf("expected backdrop glyph 88 with palette index 4, got %+v", layers[0])
	}
	if layers[1].glyphID != 77 || layers[1].paletteIndex != 3 {
		t.Fatalf("expected source glyph 77 with palette index 3, got %+v", layers[1])
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileSelectTTCFont(t *testing.T) {
	t.Parallel()

	var data []byte
	data = append(data, 't', 't', 'c', 'f')
	data = appendUint16(data, 2)  // major version
	data = appendUint16(data, 0)  // minor version
	data = appendUint32(data, 1)  // font count
	data = appendUint32(data, 24) // first face offset
	data = append(data, make([]byte, 8)...)
	data = appendUint32(data, 0x00010000)

	utf := newUTF8Font(&fileReader{array: data})
	if got := utf.readUint32(); got != 0x74746366 {
		t.Fatalf("expected TTC signature, got %#x", got)
	}
	if err := utf.selectTTCFont(); err != nil {
		t.Fatalf("expected TTC font selection to succeed, got %v", err)
	}
	if utf.fontOffset != 24 {
		t.Fatalf("expected selected font offset 24, got %d", utf.fontOffset)
	}
	if utf.fileReader.readerPosition != 24 {
		t.Fatalf("expected reader at selected font offset, got %d", utf.fileReader.readerPosition)
	}
}

func TestUTF8FontFileCBDTBitmapGlyphImage(t *testing.T) {
	t.Parallel()

	png := testPNG(13, 17)
	glyphData := []byte{12, 10, 0xff, 9, 11}
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
	glyph := utf.bitmapGlyphImage(7, 18)
	if glyph == nil {
		t.Fatal("expected CBDT bitmap glyph image")
	}
	if glyph.imageType != "png" || string(glyph.data) != string(png) {
		t.Fatalf("expected PNG data to round-trip, got type %q length %d", glyph.imageType, len(glyph.data))
	}
	if glyph.width != 10 || glyph.height != 12 || glyph.bearingX != -1 || glyph.bearingY != 9 || glyph.advance != 11 {
		t.Fatalf("unexpected glyph metrics: %+v", glyph)
	}
	if glyph.ppemX != 18 || glyph.ppemY != 18 {
		t.Fatalf("expected 18 ppem strike, got %dx%d", glyph.ppemX, glyph.ppemY)
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileCBDTBitmapGlyphImageFormat18(t *testing.T) {
	t.Parallel()

	png := testPNG(19, 23)
	glyphData := appendBigGlyphMetrics(nil, 14, 15, 2, 12, 16)
	glyphData = appendUint32(glyphData, len(png))
	glyphData = append(glyphData, png...)

	cblc := buildTestCBLCFormat18Table(7, 18, len(glyphData))
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
	glyph := utf.bitmapGlyphImage(7, 18)
	if glyph == nil {
		t.Fatal("expected CBDT format 18 bitmap glyph image")
	}
	if glyph.imageType != "png" || string(glyph.data) != string(png) {
		t.Fatalf("expected PNG data to round-trip, got type %q length %d", glyph.imageType, len(glyph.data))
	}
	if glyph.width != 15 || glyph.height != 14 || glyph.bearingX != 2 || glyph.bearingY != 12 || glyph.advance != 16 {
		t.Fatalf("unexpected glyph metrics: %+v", glyph)
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileCBDTBitmapGlyphImageFormat19(t *testing.T) {
	t.Parallel()

	png := testPNG(29, 31)
	glyphData := appendUint32(nil, len(png))
	glyphData = append(glyphData, png...)

	cblc := buildTestCBLCFormat19Table(7, 18, len(glyphData))
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
	glyph := utf.bitmapGlyphImage(7, 18)
	if glyph == nil {
		t.Fatal("expected CBDT format 19 bitmap glyph image")
	}
	if glyph.imageType != "png" || string(glyph.data) != string(png) {
		t.Fatalf("expected PNG data to round-trip, got type %q length %d", glyph.imageType, len(glyph.data))
	}
	if glyph.width != 15 || glyph.height != 14 || glyph.bearingX != -2 || glyph.bearingY != 12 || glyph.advance != 16 {
		t.Fatalf("expected metrics from CBLC format 5, got %+v", glyph)
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func TestUTF8FontFileSBIXBitmapGlyphImage(t *testing.T) {
	t.Parallel()

	sbix := buildTestSBIXTable(8, 7, []testSBIXStrike{
		{ppem: 20, ppi: 72, width: 21, height: 22, originX: -2, originY: 3},
		{ppem: 64, ppi: 72, width: 31, height: 32, originX: 4, originY: -5},
	})
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: sbix},
		tableDescriptions: map[string]*tableDescription{"sbix": {position: 0, size: len(sbix)}},
	}

	utf.parseSBIXTable(8)
	if len(utf.sbixStrikes) != 2 {
		t.Fatalf("expected two sbix strikes, got %d", len(utf.sbixStrikes))
	}
	if utf.sbixStrikes[0].ppem != 20 || utf.sbixStrikes[1].ppem != 64 {
		t.Fatalf("unexpected strike ppems: %+v", utf.sbixStrikes)
	}

	small := utf.bitmapGlyphImage(7, 18)
	if small == nil {
		t.Fatal("expected small sbix bitmap glyph image")
	}
	if small.width != 21 || small.height != 22 || small.originOffsetX != -2 || small.originOffsetY != 3 {
		t.Fatalf("unexpected small strike glyph: %+v", small)
	}

	large := utf.bitmapGlyphImage(7, 24)
	if large == nil {
		t.Fatal("expected large sbix bitmap glyph image")
	}
	if large.width != 31 || large.height != 32 || large.originOffsetX != 4 || large.originOffsetY != -5 {
		t.Fatalf("unexpected large strike glyph: %+v", large)
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}

func buildTestCMAPTable() []byte {
	var cmap []byte
	cmap = appendUint16(cmap, 0)  // version
	cmap = appendUint16(cmap, 2)  // numTables
	cmap = appendUint16(cmap, 3)  // platformID Windows
	cmap = appendUint16(cmap, 1)  // encodingID Unicode BMP
	cmap = appendUint32(cmap, 20) // subtable offset
	cmap = appendUint16(cmap, 3)  // platformID Windows
	cmap = appendUint16(cmap, 10) // encodingID Unicode full repertoire
	cmap = appendUint32(cmap, 36) // subtable offset

	// Minimal empty format-4 subtable at offset 20.
	cmap = appendUint16(cmap, 4)  // format
	cmap = appendUint16(cmap, 16) // length
	cmap = appendUint16(cmap, 0)  // language
	cmap = append(cmap, make([]byte, 10)...)

	// Format-12 subtable at offset 36 with one U+1F600 group.
	cmap = appendUint16(cmap, 12)      // format
	cmap = appendUint16(cmap, 0)       // reserved
	cmap = appendUint32(cmap, 28)      // length
	cmap = appendUint32(cmap, 0)       // language
	cmap = appendUint32(cmap, 1)       // nGroups
	cmap = appendUint32(cmap, 0x1F600) // startCharCode
	cmap = appendUint32(cmap, 0x1F600) // endCharCode
	cmap = appendUint32(cmap, 42)      // startGlyphID
	return cmap
}

func buildTestCOLRV1Table() []byte {
	var colr []byte
	colr = appendUint16(colr, 1)  // version
	colr = appendUint16(colr, 0)  // numBaseGlyphRecords
	colr = appendUint32(colr, 0)  // baseGlyphRecordsOffset
	colr = appendUint32(colr, 0)  // layerRecordsOffset
	colr = appendUint16(colr, 0)  // numLayerRecords
	colr = appendUint32(colr, 34) // baseGlyphListOffset
	colr = appendUint32(colr, 50) // layerListOffset
	colr = appendUint32(colr, 0)  // clipListOffset
	colr = appendUint32(colr, 0)  // varIndexMapOffset
	colr = appendUint32(colr, 0)  // itemVariationStoreOffset

	colr = appendUint32(colr, 1)  // BaseGlyphList record count
	colr = appendUint16(colr, 42) // base glyph ID
	colr = appendUint32(colr, 10) // PaintColrLayers offset from BaseGlyphList

	colr = append(colr, 1, 1)    // PaintColrLayers: format, numLayers
	colr = appendUint32(colr, 0) // firstLayerIndex

	colr = appendUint32(colr, 1) // LayerList layer count
	colr = appendUint32(colr, 8) // PaintGlyph offset from LayerList

	colr = append(colr, 10)           // PaintGlyph format
	colr = appendOffset24(colr, 6)    // PaintSolid offset from PaintGlyph
	colr = appendUint16(colr, 77)     // layer glyph ID
	colr = append(colr, 2)            // PaintSolid format
	colr = appendUint16(colr, 3)      // palette index
	colr = appendUint16(colr, 0x4000) // alpha
	return colr
}

func buildTestCOLRV1ReuseTable() []byte {
	var colr []byte
	colr = appendCOLRV1Header(colr, 34, 59)

	colr = appendUint32(colr, 2)  // BaseGlyphList record count
	colr = appendUint16(colr, 42) // base glyph ID that reuses glyph 43
	colr = appendUint32(colr, 16) // PaintColrGlyph offset from BaseGlyphList
	colr = appendUint16(colr, 43) // reused base glyph ID
	colr = appendUint32(colr, 19) // PaintColrLayers offset from BaseGlyphList
	colr = append(colr, 11)       // PaintColrGlyph format
	colr = appendUint16(colr, 43) // reused base glyph ID

	colr = append(colr, 1, 1)    // PaintColrLayers: format, numLayers
	colr = appendUint32(colr, 0) // firstLayerIndex

	colr = appendUint32(colr, 1) // LayerList layer count
	colr = appendUint32(colr, 8) // PaintGlyph offset from LayerList
	colr = appendPaintGlyphSolid(colr, 77, 3)
	return colr
}

func buildTestCOLRV1CompositeTable() []byte {
	var colr []byte
	colr = appendCOLRV1Header(colr, 34, 74)

	colr = appendUint32(colr, 1)  // BaseGlyphList record count
	colr = appendUint16(colr, 42) // base glyph ID
	colr = appendUint32(colr, 10) // PaintComposite offset from BaseGlyphList

	colr = append(colr, 32)         // PaintComposite format
	colr = appendOffset24(colr, 8)  // source PaintGlyph offset from PaintComposite
	colr = append(colr, 3)          // COMPOSITE_SRC_OVER
	colr = appendOffset24(colr, 19) // backdrop PaintGlyph offset from PaintComposite
	colr = appendPaintGlyphSolid(colr, 77, 3)
	colr = appendPaintGlyphSolid(colr, 88, 4)

	colr = appendUint32(colr, 0) // empty LayerList, unused by this paint graph
	return colr
}

func appendCOLRV1Header(dst []byte, baseGlyphListOffset, layerListOffset int) []byte {
	dst = appendUint16(dst, 1)                   // version
	dst = appendUint16(dst, 0)                   // numBaseGlyphRecords
	dst = appendUint32(dst, 0)                   // baseGlyphRecordsOffset
	dst = appendUint32(dst, 0)                   // layerRecordsOffset
	dst = appendUint16(dst, 0)                   // numLayerRecords
	dst = appendUint32(dst, baseGlyphListOffset) // baseGlyphListOffset
	dst = appendUint32(dst, layerListOffset)     // layerListOffset
	dst = appendUint32(dst, 0)                   // clipListOffset
	dst = appendUint32(dst, 0)                   // varIndexMapOffset
	return appendUint32(dst, 0)                  // itemVariationStoreOffset
}

func appendPaintGlyphSolid(dst []byte, glyphID, paletteIndex int) []byte {
	dst = append(dst, 10)        // PaintGlyph format
	dst = appendOffset24(dst, 6) // PaintSolid offset from PaintGlyph
	dst = appendUint16(dst, glyphID)
	dst = append(dst, 2) // PaintSolid format
	dst = appendUint16(dst, paletteIndex)
	return appendUint16(dst, 0x4000) // alpha
}

func appendUint16(dst []byte, value int) []byte {
	return append(dst, byte(value>>8), byte(value))
}

func appendUint32(dst []byte, value int) []byte {
	return append(dst, byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
}

func appendOffset24(dst []byte, value int) []byte {
	return append(dst, byte(value>>16), byte(value>>8), byte(value))
}

func buildTestCBLCFormat17Table(glyphID, ppem, glyphDataLength int) []byte {
	return buildTestCBLCFormat1Table(glyphID, ppem, 17, glyphDataLength)
}

func buildTestCBLCFormat18Table(glyphID, ppem, glyphDataLength int) []byte {
	return buildTestCBLCFormat3Table(glyphID, ppem, 18, glyphDataLength)
}

func buildTestCBLCFormat1Table(glyphID, ppem, imageFormat, glyphDataLength int) []byte {
	var cblc []byte
	cblc = appendUint32(cblc, 0x00030000) // version
	cblc = appendUint32(cblc, 1)          // strike count

	cblc = appendUint32(cblc, 56) // indexSubtableListOffset
	cblc = appendUint32(cblc, 24) // indexTablesSize
	cblc = appendUint32(cblc, 1)  // numberOfIndexSubtables
	cblc = appendUint32(cblc, 0)  // colorRef
	cblc = append(cblc, make([]byte, 24)...)
	cblc = appendUint16(cblc, glyphID) // startGlyphIndex
	cblc = appendUint16(cblc, glyphID) // endGlyphIndex
	cblc = append(cblc, byte(ppem), byte(ppem), 32, 1)

	cblc = appendUint16(cblc, glyphID)
	cblc = appendUint16(cblc, glyphID)
	cblc = appendUint32(cblc, 8) // index subtable offset

	cblc = appendUint16(cblc, 1)           // indexFormat
	cblc = appendUint16(cblc, imageFormat) // imageFormat
	cblc = appendUint32(cblc, 4)           // imageDataOffset, after CBDT version
	cblc = appendUint32(cblc, 0)
	return appendUint32(cblc, glyphDataLength)
}

func buildTestCBLCFormat3Table(glyphID, ppem, imageFormat, glyphDataLength int) []byte {
	cblc := buildTestCBLCHeader(glyphID, ppem)
	cblc = appendUint16(cblc, 3)           // indexFormat
	cblc = appendUint16(cblc, imageFormat) // imageFormat
	cblc = appendUint32(cblc, 4)           // imageDataOffset, after CBDT version
	cblc = appendUint16(cblc, 0)
	return appendUint16(cblc, glyphDataLength)
}

func buildTestCBLCFormat19Table(glyphID, ppem, glyphDataLength int) []byte {
	cblc := buildTestCBLCHeader(glyphID, ppem)
	cblc = appendUint16(cblc, 5)  // indexFormat
	cblc = appendUint16(cblc, 19) // imageFormat
	cblc = appendUint32(cblc, 4)  // imageDataOffset, after CBDT version
	cblc = appendUint32(cblc, glyphDataLength)
	cblc = appendBigGlyphMetrics(cblc, 14, 15, -2, 12, 16)
	cblc = appendUint32(cblc, 1) // numGlyphs
	return appendUint16(cblc, glyphID)
}

func buildTestCBLCHeader(glyphID, ppem int) []byte {
	var cblc []byte
	cblc = appendUint32(cblc, 0x00030000) // version
	cblc = appendUint32(cblc, 1)          // strike count

	cblc = appendUint32(cblc, 56) // indexSubtableListOffset
	cblc = appendUint32(cblc, 32) // indexTablesSize
	cblc = appendUint32(cblc, 1)  // numberOfIndexSubtables
	cblc = appendUint32(cblc, 0)  // colorRef
	cblc = append(cblc, make([]byte, 24)...)
	cblc = appendUint16(cblc, glyphID) // startGlyphIndex
	cblc = appendUint16(cblc, glyphID) // endGlyphIndex
	cblc = append(cblc, byte(ppem), byte(ppem), 32, 1)

	cblc = appendUint16(cblc, glyphID)
	cblc = appendUint16(cblc, glyphID)
	return appendUint32(cblc, 8) // index subtable offset
}

func appendBigGlyphMetrics(dst []byte, height, width, bearingX, bearingY, advance int) []byte {
	return append(dst, byte(height), byte(width), byte(bearingX), byte(bearingY), byte(advance), 0, 0, 0)
}

type testSBIXStrike struct {
	ppem    int
	ppi     int
	width   int
	height  int
	originX int
	originY int
}

func buildTestSBIXTable(numGlyphs, glyphID int, strikes []testSBIXStrike) []byte {
	var sbix []byte
	sbix = appendUint16(sbix, 1)            // version
	sbix = appendUint16(sbix, 0)            // flags
	sbix = appendUint32(sbix, len(strikes)) // strike count

	offsetsPosition := len(sbix)
	for range strikes {
		sbix = appendUint32(sbix, 0)
	}

	strikeOffsets := make([]int, len(strikes))
	for i, strike := range strikes {
		strikeOffsets[i] = len(sbix)
		sbix = append(sbix, buildTestSBIXStrike(numGlyphs, glyphID, strike)...)
	}

	for i, offset := range strikeOffsets {
		copy(sbix[offsetsPosition+i*4:], appendUint32(nil, offset))
	}
	return sbix
}

func buildTestSBIXStrike(numGlyphs, glyphID int, strike testSBIXStrike) []byte {
	png := testPNG(strike.width, strike.height)
	glyphData := appendInt16(nil, strike.originX)
	glyphData = appendInt16(glyphData, strike.originY)
	glyphData = append(glyphData, 'p', 'n', 'g', ' ')
	glyphData = append(glyphData, png...)

	glyphDataOffset := 4 + (numGlyphs+1)*4
	var data []byte
	data = appendUint16(data, strike.ppem)
	data = appendUint16(data, strike.ppi)
	for i := 0; i <= numGlyphs; i++ {
		offset := 0
		if i == glyphID {
			offset = glyphDataOffset
		}
		if i == glyphID+1 {
			offset = glyphDataOffset + len(glyphData)
		}
		data = appendUint32(data, offset)
	}
	return append(data, glyphData...)
}

func testPNG(width, height int) []byte {
	png := []byte("\x89PNG\r\n\x1a\n")
	png = appendUint32(png, 13) // IHDR length
	png = append(png, 'I', 'H', 'D', 'R')
	png = appendUint32(png, width)
	png = appendUint32(png, height)
	return append(png, 8, 6, 0, 0, 0)
}

func appendInt16(dst []byte, value int) []byte {
	return append(dst, byte(value>>8), byte(value))
}
