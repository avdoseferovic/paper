package pdf

import "testing"

func TestHasOutlineTables(t *testing.T) {
	t.Parallel()

	utf := &utf8FontFile{tableDescriptions: map[string]*tableDescription{
		"glyf": {}, "loca": {},
	}}
	if !utf.hasOutlineTables() {
		t.Fatal("expected font with glyf and loca tables to report outline tables")
	}

	utf = &utf8FontFile{tableDescriptions: map[string]*tableDescription{"glyf": {}}}
	if utf.hasOutlineTables() {
		t.Fatal("expected font without loca table to report no outline tables")
	}
}

func TestSetFallbackOS2Metrics(t *testing.T) {
	t.Parallel()

	utf := &utf8FontFile{bbox: fontBoxType{Ymin: -200, Ymax: 800}}
	utf.setFallbackOS2Metrics(1.0)
	if utf.ascent != 800 || utf.descent != -200 || utf.capHeight != 800 {
		t.Fatalf("expected metrics derived from the bounding box, got ascent=%d descent=%d capHeight=%d",
			utf.ascent, utf.descent, utf.capHeight)
	}

	utf = &utf8FontFile{ascent: 700, descent: -150, bbox: fontBoxType{Ymin: -200, Ymax: 800}}
	utf.setFallbackOS2Metrics(1.0)
	if utf.ascent != 700 || utf.descent != -150 || utf.capHeight != 700 {
		t.Fatalf("expected existing hhea metrics to be preserved, got ascent=%d descent=%d capHeight=%d",
			utf.ascent, utf.descent, utf.capHeight)
	}
}

func TestParseLOCATableFormats(t *testing.T) {
	t.Parallel()

	t.Run("format0", func(t *testing.T) {
		var loca []byte
		for _, v := range []int{5, 10, 15} {
			loca = appendUint16(loca, v)
		}
		utf := &utf8FontFile{
			fileReader:        &fileReader{array: loca},
			tableDescriptions: map[string]*tableDescription{"loca": {position: 0, size: len(loca)}},
		}
		utf.parseLOCATable(0, 2)
		if utf.err != nil {
			t.Fatalf("expected no error, got %v", utf.err)
		}
		want := []int{10, 20, 30}
		for i, v := range want {
			if utf.symbolPosition[i] != v {
				t.Fatalf("expected doubled short offsets %v, got %v", want, utf.symbolPosition)
			}
		}
	})

	t.Run("format1", func(t *testing.T) {
		var loca []byte
		for _, v := range []int{7, 14, 21} {
			loca = appendUint32(loca, v)
		}
		utf := &utf8FontFile{
			fileReader:        &fileReader{array: loca},
			tableDescriptions: map[string]*tableDescription{"loca": {position: 0, size: len(loca)}},
		}
		utf.parseLOCATable(1, 2)
		if utf.err != nil {
			t.Fatalf("expected no error, got %v", utf.err)
		}
		want := []int{7, 14, 21}
		for i, v := range want {
			if utf.symbolPosition[i] != v {
				t.Fatalf("expected long offsets %v, got %v", want, utf.symbolPosition)
			}
		}
	})

	t.Run("unknownFormat", func(t *testing.T) {
		utf := &utf8FontFile{
			fileReader:        &fileReader{array: make([]byte, 16)},
			tableDescriptions: map[string]*tableDescription{"loca": {position: 0, size: 16}},
		}
		utf.parseLOCATable(7, 2)
		if utf.err == nil {
			t.Fatal("expected error for unknown loca format")
		}
	})
}

func buildTestCPALTable(records []colorRecord) []byte {
	var cpal []byte
	cpal = appendUint16(cpal, 0)            // version
	cpal = appendUint16(cpal, len(records)) // numPaletteEntries
	cpal = appendUint16(cpal, 1)            // numPalettes
	cpal = appendUint16(cpal, len(records)) // numColorRecords
	cpal = appendUint32(cpal, 16)           // colorRecordsArrayOffset
	cpal = appendUint32(cpal, 0)            // colorRecordIndices (one palette)
	for _, rec := range records {
		cpal = append(cpal, rec.b, rec.g, rec.r, rec.a)
	}
	return cpal
}

func TestParseCPALTableAndPaletteColor(t *testing.T) {
	t.Parallel()

	cpal := buildTestCPALTable([]colorRecord{
		{r: 3, g: 2, b: 1, a: 4},
		{r: 7, g: 6, b: 5, a: 8},
	})
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: cpal},
		tableDescriptions: map[string]*tableDescription{"CPAL": {position: 0, size: len(cpal)}},
	}

	utf.parseCPALTable()
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
	if got := utf.paletteColor(1); got.r != 7 || got.g != 6 || got.b != 5 || got.a != 8 {
		t.Fatalf("unexpected palette color: %+v", got)
	}
	if got := utf.paletteColor(9); got != (colorRecord{a: 255}) {
		t.Fatalf("expected opaque black for out-of-range index, got %+v", got)
	}

	missing := &utf8FontFile{tableDescriptions: map[string]*tableDescription{}}
	missing.parseCPALTable()
	if missing.cpalTable != nil {
		t.Fatal("expected no CPAL table when the font has none")
	}
	if got := missing.paletteColor(0); got != (colorRecord{a: 255}) {
		t.Fatalf("expected opaque black without CPAL table, got %+v", got)
	}
}

func buildTestCBLCFormat2Table(glyphID, ppem, imageSize int) []byte {
	cblc := buildTestCBLCHeader(glyphID, ppem)
	cblc = appendUint16(cblc, 2)  // indexFormat
	cblc = appendUint16(cblc, 17) // imageFormat
	cblc = appendUint32(cblc, 4)  // imageDataOffset, after CBDT version
	cblc = appendUint32(cblc, imageSize)
	return appendBigGlyphMetrics(cblc, 14, 15, 2, 12, 16)
}

func buildTestCBLCFormat4Table(glyphID, ppem, glyphDataLength int) []byte {
	cblc := buildTestCBLCHeader(glyphID, ppem)
	cblc = appendUint16(cblc, 4)  // indexFormat
	cblc = appendUint16(cblc, 17) // imageFormat
	cblc = appendUint32(cblc, 4)  // imageDataOffset, after CBDT version
	cblc = appendUint32(cblc, 1)  // numGlyphs (pairs)
	cblc = appendUint16(cblc, glyphID)
	cblc = appendUint16(cblc, 0) // offset of first glyph
	cblc = appendUint16(cblc, glyphID+1)
	return appendUint16(cblc, glyphDataLength) // end offset
}

func cbdtFormat17GlyphData(t *testing.T) []byte {
	t.Helper()
	png := testPNG(13, 17)
	glyphData := []byte{12, 10, 0xff, 9, 11}
	glyphData = appendUint32(glyphData, len(png))
	return append(glyphData, png...)
}

func TestParseCBLCFormat2BitmapGlyph(t *testing.T) {
	t.Parallel()

	glyphData := cbdtFormat17GlyphData(t)
	cblc := buildTestCBLCFormat2Table(7, 18, len(glyphData))
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
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
	glyph := utf.bitmapGlyphImage(7, 18)
	if glyph == nil {
		t.Fatal("expected format-2 indexed bitmap glyph")
	}
	if glyph.imageType != "png" {
		t.Fatalf("expected PNG glyph image, got %q", glyph.imageType)
	}
}

func TestParseCBLCFormat4BitmapGlyph(t *testing.T) {
	t.Parallel()

	glyphData := cbdtFormat17GlyphData(t)
	cblc := buildTestCBLCFormat4Table(7, 18, len(glyphData))
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
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
	glyph := utf.bitmapGlyphImage(7, 18)
	if glyph == nil {
		t.Fatal("expected format-4 indexed bitmap glyph")
	}
	if glyph.imageType != "png" {
		t.Fatalf("expected PNG glyph image, got %q", glyph.imageType)
	}
}

// buildTestCOLRV0Table builds a version-0 COLR table with explicit base glyph
// and layer records.
func buildTestCOLRV0Table() []byte {
	var colr []byte
	colr = appendUint16(colr, 0)  // version
	colr = appendUint16(colr, 1)  // numBaseGlyphRecords
	colr = appendUint32(colr, 14) // baseGlyphRecordsOffset
	colr = appendUint32(colr, 20) // layerRecordsOffset
	colr = appendUint16(colr, 2)  // numLayerRecords

	colr = appendUint16(colr, 42) // base glyph ID
	colr = appendUint16(colr, 0)  // first layer index
	colr = appendUint16(colr, 2)  // layer count

	colr = appendUint16(colr, 77) // layer glyph ID
	colr = appendUint16(colr, 3)  // palette index
	colr = appendUint16(colr, 78) // layer glyph ID
	colr = appendUint16(colr, 4)  // palette index
	return colr
}

func TestParseCOLRTableVersion0Records(t *testing.T) {
	t.Parallel()

	colr := buildTestCOLRV0Table()
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: colr},
		tableDescriptions: map[string]*tableDescription{"COLR": {position: 0, size: len(colr)}},
	}

	utf.parseCOLRTable()
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
	layers := utf.colorGlyphLayers(42)
	if len(layers) != 2 {
		t.Fatalf("expected two v0 layers, got %v", layers)
	}
	if layers[0].glyphID != 77 || layers[0].paletteIndex != 3 {
		t.Fatalf("unexpected first layer: %+v", layers[0])
	}
	if layers[1].glyphID != 78 || layers[1].paletteIndex != 4 {
		t.Fatalf("unexpected second layer: %+v", layers[1])
	}
	if utf.colorGlyphLayers(99) != nil {
		t.Fatal("expected no layers for unknown glyph in a v0 table")
	}
}

func TestParseCOLRV1ColorEdgeCases(t *testing.T) {
	t.Parallel()

	// Format 3 (variable solid) reads the palette index directly.
	data := append([]byte{3}, appendUint16(nil, 6)...)
	utf := &utf8FontFile{fileReader: &fileReader{array: data}}
	if got := utf.parseCOLRV1Color(0, 0); got != 0xFFFF {
		// offset 0 is rejected before any read.
		t.Fatalf("expected sentinel for non-positive offset, got %d", got)
	}

	data = append([]byte{0}, data...) // shift the paint to offset 1
	utf = &utf8FontFile{fileReader: &fileReader{array: data}}
	if got := utf.parseCOLRV1Color(1, 0); got != 6 {
		t.Fatalf("expected palette index 6 from format-3 paint, got %d", got)
	}

	// Unknown paint format yields the sentinel index.
	utf = &utf8FontFile{fileReader: &fileReader{array: []byte{0, 99, 0, 0}}}
	if got := utf.parseCOLRV1Color(1, 0); got != 0xFFFF {
		t.Fatalf("expected sentinel for unknown paint format, got %d", got)
	}

	// Depth guard stops recursion.
	utf = &utf8FontFile{fileReader: &fileReader{array: []byte{0, 2, 0, 6}}}
	if got := utf.parseCOLRV1Color(1, 17); got != 0xFFFF {
		t.Fatalf("expected sentinel at recursion limit, got %d", got)
	}

	// Transform paints (formats 12-31) follow the nested paint offset.
	var transform []byte
	transform = append(transform, 0)            // padding so offsets are positive
	transform = append(transform, 12)           // PaintTransform format
	transform = appendOffset24(transform, 4)    // nested paint offset
	transform = append(transform, 2)            // PaintSolid format
	transform = appendUint16(transform, 5)      // palette index
	transform = appendUint16(transform, 0x4000) // alpha
	utf = &utf8FontFile{fileReader: &fileReader{array: transform}}
	if got := utf.parseCOLRV1Color(1, 0); got != 5 {
		t.Fatalf("expected palette index 5 through transform paint, got %d", got)
	}
}

// buildTestCOLRV1GradientTable builds a COLRv1 table whose layer paint is a
// PaintLinearGradient, so the palette index is resolved from the first color
// line stop.
func buildTestCOLRV1GradientTable() []byte {
	var colr []byte
	colr = appendCOLRV1Header(colr, 34, 50)

	colr = appendUint32(colr, 1)  // BaseGlyphList record count
	colr = appendUint16(colr, 42) // base glyph ID
	colr = appendUint32(colr, 10) // PaintColrLayers offset from BaseGlyphList

	colr = append(colr, 1, 1)    // PaintColrLayers: format, numLayers
	colr = appendUint32(colr, 0) // firstLayerIndex

	colr = appendUint32(colr, 1) // LayerList layer count
	colr = appendUint32(colr, 8) // PaintGlyph offset from LayerList

	colr = append(colr, 10)        // PaintGlyph format
	colr = appendOffset24(colr, 6) // PaintLinearGradient offset from PaintGlyph
	colr = appendUint16(colr, 77)  // layer glyph ID

	colr = append(colr, 4)         // PaintLinearGradient format
	colr = appendOffset24(colr, 4) // ColorLine offset from PaintLinearGradient

	colr = append(colr, 0)       // ColorLine extend
	colr = appendUint16(colr, 1) // stop count
	colr = appendUint16(colr, 0) // stop offset
	colr = appendUint16(colr, 9) // stop palette index
	return colr
}

func TestUTF8FontFileColorGlyphLayersV1GradientPaint(t *testing.T) {
	t.Parallel()

	colr := buildTestCOLRV1GradientTable()
	utf := &utf8FontFile{
		fileReader:        &fileReader{array: colr},
		tableDescriptions: map[string]*tableDescription{"COLR": {position: 0, size: len(colr)}},
	}

	utf.parseCOLRTable()
	layers := utf.colorGlyphLayers(42)
	if len(layers) != 1 {
		t.Fatalf("expected one gradient layer, got %v", layers)
	}
	if layers[0].glyphID != 77 || layers[0].paletteIndex != 9 {
		t.Fatalf("expected glyph 77 with first gradient stop palette index 9, got %+v", layers[0])
	}
	if utf.err != nil {
		t.Fatalf("expected no parse error, got %v", utf.err)
	}
}
